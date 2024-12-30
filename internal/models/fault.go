package models

type SOAPFault struct {
	Envelope struct {
		Body struct {
			Fault struct {
				Code    string `xml:"faultcode"`
				Message string `xml:"faultstring"`
				Detail  struct {
					ErrorCode    string `xml:"errorCode"`
					ErrorMessage string `xml:"errorMessage"`
				} `xml:"detail"`
			} `xml:"Fault"`
		} `xml:"Body"`
	} `xml:"Envelope"`
}

const (
	ErrorRecordNotFoundCode          = "404"
	ErrorRecordNotFoundMessage       = "Запись не найдена"
	ErrorRecordNotFoundDetail        = "Запрашиваемая запись отсутствует в базе данных."
	ErrorRecordEmailExistsCode       = "409"
	ErrorRecordEmailExistsMessage    = "Запись уже существует"
	ErrorRecordEmailExistsDetail     = "Запись с данным email уже существует"
	ErrorEmailIncorrectCode          = "409"
	ErrorEmailIncorrectMessage       = "Некорректный email"
	ErrorEmailIncorrectDetail        = "Получен некорректный email"
	ErrorPhoneNumberIncorrectCode    = "409"
	ErrorPhoneNumberIncorrectMessage = "Некорректный phone number"
	ErrorPhoneNumberIncorrectDetail  = "Получен некорректный phone number"
	ErrorAuthIncorrectCode           = "401"
	ErrorAuthIncorrectMessage        = "Неудачная Аутентификация"
	ErrorAuthIncorrectDetail         = "Введен некорректный логин или пароль"
)
