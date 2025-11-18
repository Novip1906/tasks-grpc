package email

import (
	"bytes"
	"html/template"
)

const verificationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        .container { 
            max-width: 600px; 
            margin: 0 auto; 
            padding: 20px; 
            font-family: Arial, sans-serif; 
            border: 1px solid #e0e0e0;
            border-radius: 8px;
        }
        .code { 
            font-size: 32px; 
            font-weight: bold; 
            color: #2563eb; 
            text-align: center;
            margin: 20px 0;
            padding: 10px;
            background: #f8fafc;
            border-radius: 4px;
        }
        .footer {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #e0e0e0;
            color: #666;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h2>Здравствуйте, {{.Username}}!</h2>
        <p>Для завершения регистрации введите следующий код подтверждения:</p>
        <div class="code">{{.Code}}</div>
        <p>Код действителен в течение 15 минут.</p>
        <div class="footer">
            <p>Если вы не запрашивали это письмо, просто проигнорируйте его.</p>
        </div>
    </div>
</body>
</html>`

func (s *EmailSenderService) renderVerificationTemplate(username, code string) (string, error) {
	tmpl := template.Must(template.New("verification").Parse(verificationTemplate))

	var buf bytes.Buffer
	data := struct {
		Username string
		Code     string
	}{
		Username: username,
		Code:     code,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
