// apiGateway/internal/http-server/resp/resp.go
package resp

import "net/http"

// Response - общая структура для всех ответов API
type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // Используется для успешных ответов
}

// OK - создает успешный ответ
func OK(message string, data interface{}) Response {
	return Response{
		Status:  http.StatusOK, // 200 OK
		Message: message,
		Data:    data,
	}
}

// Error - создает ответ об ошибке
func Error(message string) Response {
	return Response{
		Status:  http.StatusInternalServerError, // Можно использовать другие статусы для разных ошибок
		Message: message,
	}
}

// Вы можете добавить другие функции, например:
// func BadRequest(message string) Response {
//     return Response{
//         Status:  http.StatusBadRequest, // 400 Bad Request
//         Message: message,
//     }
// }
