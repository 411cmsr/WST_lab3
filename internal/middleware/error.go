package middleware

import (
	"WST_lab1_server_new1/internal/models"

	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandler - middleware для обработки ошибок
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Type:     "/errors/internal-server-error",
					Title:    "Internal Server Error",
					Status:   http.StatusInternalServerError,
					Detail:   "An unexpected error occurred.",
					Instance: c.Request.RequestURI,
				})
			}
		}()
		c.Next()

		// Обработка ошибок после выполнения запроса
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				var status int
				var title string

				// Определяем тип ошибки и соответствующий статус
				switch err.Type {
				case gin.ErrorTypePrivate:
					status = http.StatusInternalServerError
					title = "Internal Server Error"
				case gin.ErrorTypePublic:
					status = http.StatusBadRequest
					title = "Bad Request"
				default:
					status = http.StatusInternalServerError
					title = "Internal Server Error"
				}

				// Возвращаем ответ с ошибкой
				c.JSON(status, models.ErrorResponse{
					Type:     "/errors/" + title,
					Title:    title,
					Status:   status,
					Detail:   err.Error(),
					Instance: c.Request.RequestURI,
				})
				return
			}
		}
	}
}
