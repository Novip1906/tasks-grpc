package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	es "github.com/elastic/go-elasticsearch/v9"

	"github.com/Novip1906/tasks-grpc/tasks/internal/models"
)

type Client struct {
	es    *es.Client
	index string
	log   *slog.Logger
}

func NewClient(addresses []string, index string, log *slog.Logger) (*Client, error) {
	cfg := es.Config{
		Addresses: addresses,
	}
	c, err := es.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}

	client := &Client{
		es:    c,
		index: index,
		log:   log,
	}

	if err := client.ensureIndex(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) Search(ctx context.Context, userId int64, query string) ([]*models.Task, error) {
	body := fmt.Sprintf(`
{
  "query": {
    "bool": {
      "must": [
        { "match": { "text": %q } }
      ],
      "filter": [
        { "term": { "user_id": %d } }
      ]
    }
  },
  "sort": [
    { "created_at": { "order": "desc" } }
  ]
}`, query, userId)

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.index),
		c.es.Search.WithBody(strings.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("es search error: %s", res.String())
	}

	var raw struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Id         int64     `json:"id"`
					Text       string    `json:"text"`
					AuthorName string    `json:"author_name"`
					CreatedAt  time.Time `json:"created_at"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	tasks := make([]*models.Task, 0, len(raw.Hits.Hits))
	for _, h := range raw.Hits.Hits {
		tasks = append(tasks, &models.Task{
			Id:         h.Source.Id,
			Text:       h.Source.Text,
			AuthorName: h.Source.AuthorName,
			CreatedAt:  h.Source.CreatedAt,
		})
	}

	return tasks, nil
}

func (c *Client) IndexTask(ctx context.Context, task *models.Task) error {
	body := fmt.Sprintf(`
{
  "id": %d,
  "user_id": %d,
  "text": %q,
  "author_name": %q,
  "created_at": %q
}`, task.Id, task.AuthorId, task.Text, task.AuthorName, task.CreatedAt.Format(time.RFC3339))

	res, err := c.es.Index(
		c.index,
		strings.NewReader(body),
		c.es.Index.WithContext(ctx),
		c.es.Index.WithDocumentID(fmt.Sprint(task.Id)),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("es index error: %s", res.String())
	}

	return nil
}

func (c *Client) DeleteTask(ctx context.Context, taskId int64) error {
	res, err := c.es.Delete(
		c.index,
		fmt.Sprint(taskId),
		c.es.Delete.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("es delete error: %s", res.String())
	}
	return nil
}

func (c *Client) ensureIndex() error {
	res, err := c.es.Indices.Exists([]string{c.index})
	if err != nil {
		return fmt.Errorf("check index exists: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		c.log.Info("elasticsearch index exists", "index", c.index)
		return nil
	}

	if res.StatusCode != 404 {
		return fmt.Errorf("unexpected status checking index: %s", res.String())
	}

	c.log.Info("creating elasticsearch index", "index", c.index)

	mapping := `
{
  "mappings": {
    "properties": {
      "id": { "type": "long" },
	  "user_id": { "type": "long" },
      "text": { "type": "text" },
      "author_name": { "type": "keyword" },
      "created_at": { "type": "date" }
    }
  }
}`

	createRes, err := c.es.Indices.Create(
		c.index,
		c.es.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("create index error: %s", createRes.String())
	}

	c.log.Info("elasticsearch index created", "index", c.index)
	return nil
}
