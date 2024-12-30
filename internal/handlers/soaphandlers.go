package handlers

import (
	"WST_lab1_server_new1/internal/database"
	"WST_lab1_server_new1/internal/database/postgres"
	"WST_lab1_server_new1/internal/logging"
	"WST_lab1_server_new1/internal/models"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
Структура обработчика для разделения логики обработки запросов от доступа к данным
*/
type StorageHandler struct {
	Storage *postgres.Storage
}

func createSOAPFault(code string, message string, errorCode string, errorMessage string) models.SOAPFault {
	fault := models.SOAPFault{}
	fault.Envelope.Body.Fault.Code = code
	fault.Envelope.Body.Fault.Message = message
	fault.Envelope.Body.Fault.Detail.ErrorCode = errorCode
	fault.Envelope.Body.Fault.Detail.ErrorMessage = errorMessage
	return fault
}

/*
Функция проверки email на корректность
*/
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return true
}

/*
Функция проверки телефона на корректность
*/
func validatePhone(phone string) bool {
	re := regexp.MustCompile(`^\+7\d{10}$`)
	return re.MatchString(phone)
}

///////////////////////////////////////////////////////////////////////////////

func (h *StorageHandler) BasicAuth(c *gin.Context) bool {
	auth := c.Request.Header.Get("Authorization")
	if auth == "" {
		c.String(http.StatusUnauthorized, "Authorization header is missing")
		return false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		c.String(http.StatusUnauthorized, "Authorization header must start with 'Basic'")
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		c.String(http.StatusUnauthorized, "Invalid base64 encoding")
		return false
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		c.String(http.StatusUnauthorized, "Invalid authorization format")
		return false
	}

	username, password := pair[0], pair[1]
	if !validateCredentials(username, password) { 
		c.String(http.StatusUnauthorized, "Invalid username or password")
		return false
	}
	return true
}


func validateCredentials(username, password string) bool {
	const validUsername = "root"
	const validPassword = "password"
	return username == validUsername && password == validPassword
}

//////////////////////////////////////////////////////////////////////////////

// Обработчик SOAP запросов
func (sh *StorageHandler) SOAPHandler(c *gin.Context) {

	var envelope models.Envelope

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading request body")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	if err := xml.Unmarshal(body, &envelope); err != nil {
		fmt.Println("Error decoding XML:", err)
		c.String(http.StatusBadRequest, "Invalid request")
		return
	}

	fmt.Printf("Decoded Envelope: %+v\n", envelope)

	switch {
	case envelope.Body.AddPerson != nil:
		sh.addPersonHandler(c, envelope.Body.AddPerson)
	case envelope.Body.DeletePerson != nil:
		sh.deletePersonHandler(c, envelope.Body.DeletePerson)
	case envelope.Body.UpdatePerson != nil:
		sh.updatePersonHandler(c, envelope.Body.UpdatePerson)
	case envelope.Body.GetPerson != nil:
		sh.getPersonHandler(c, envelope.Body.GetPerson)
	case envelope.Body.GetAllPersons != nil:
		sh.getAllPersonsHandler(c)
	case envelope.Body.SearchPerson != nil:
		sh.searchPersonHandler(c, envelope.Body.SearchPerson)
	default:
		fmt.Println("Unsupported action")
		c.String(http.StatusBadRequest, "Unsupported action")
		return
	}
}

// Метод добавления новой записи в базу данных
func (h *StorageHandler) addPersonHandler(c *gin.Context, request *models.AddPersonRequest) {
	//////////////////////////////////////
	if !h.BasicAuth(c) {
		logging.Logger.Error("Error Invalid user login or password")
		fault := createSOAPFault("soap:Client", models.ErrorAuthIncorrectMessage, models.ErrorAuthIncorrectCode, models.ErrorAuthIncorrectDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusUnauthorized, fault)

		return
	}
	//////////////////////////////////////
	// Создаем person с данными из запроса
	person := models.Person{
		Name:      request.Name,
		Surname:   request.Surname,
		Age:       request.Age,
		Email:     request.Email,
		Telephone: request.Telephone,
	}
	if !validateEmail(person.Email) {
		logging.Logger.Info("Email is", zap.String("Email:", request.Email))
		fault := createSOAPFault("soap:Client", models.ErrorEmailIncorrectMessage, models.ErrorEmailIncorrectCode, models.ErrorEmailIncorrectDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusConflict, fault)
		return
	}

	if !validatePhone(person.Telephone) {
		logging.Logger.Info("Email exists", zap.String("Email:", request.Email))
		fault := createSOAPFault("soap:Client", models.ErrorPhoneNumberIncorrectMessage, models.ErrorPhoneNumberIncorrectCode, models.ErrorPhoneNumberIncorrectDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusConflict, fault)
		return
	}

	// Добавляем person в базу данных
	id, err := h.Storage.PersonRepository.AddPerson(&person)
	if err != nil {
		if errors.Is(err, database.ErrEmailExists) {
			logging.Logger.Info("Email exists", zap.String("Email:", request.Email), zap.Error(err))
			fault := createSOAPFault("soap:Client", models.ErrorRecordEmailExistsMessage, models.ErrorRecordEmailExistsCode, models.ErrorRecordEmailExistsDetail)
			fmt.Printf("Response Fault: %+v\n", fault)
			c.XML(http.StatusConflict, fault)
			return
		}
		fmt.Printf("Error adding person: %v\n", err)

		// Формируем SOAP Fault для ошибки добавления
		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}
	fmt.Printf("Person added with ID: %d\n", id)

	response := models.AddPersonResponse{
		ID: id,
	}

	// Возвращаем успешный ответ в формате XML
	fmt.Printf("Response: %+v\n", response)
	c.XML(http.StatusOK, response)
}

// Метод обновления записи в базе данных
func (h *StorageHandler) updatePersonHandler(c *gin.Context, request *models.UpdatePersonRequest) {
	//////////////////////////////////////
	if !h.BasicAuth(c) {
		logging.Logger.Error("Error Invalid user login or password")
		fault := createSOAPFault("soap:Client", models.ErrorAuthIncorrectMessage, models.ErrorAuthIncorrectCode, models.ErrorAuthIncorrectDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusUnauthorized, fault)
		return
	}
	//////////////////////////////////////
	//Проверяем на корректность Email
	if !validateEmail(request.Email) {
		logging.Logger.Info("Email is", zap.String("Email:", request.Email))
		fault := createSOAPFault("soap:Client", models.ErrorEmailIncorrectMessage, models.ErrorEmailIncorrectCode, models.ErrorEmailIncorrectDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusConflict, fault)
		return
	}
	//Проверяем на корректность номер телефона
	if !validatePhone(request.Telephone) {
		logging.Logger.Info("Email exists", zap.String("Email:", request.Email))
		fault := createSOAPFault("soap:Client", models.ErrorPhoneNumberIncorrectMessage, models.ErrorPhoneNumberIncorrectCode, models.ErrorPhoneNumberIncorrectDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusConflict, fault)
		return
	}
	// Проверяем, существует ли запись с данным ID
	checkByID, err := h.Storage.PersonRepository.CheckPersonByID(uint(request.ID))
	if !checkByID {
		fault := createSOAPFault("soap:Client", models.ErrorRecordNotFoundMessage, models.ErrorRecordNotFoundCode, models.ErrorRecordNotFoundDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusNotFound, fault)
		return
	}
	if err != nil {
		logging.Logger.Error("Error getting person with ID", zap.Uint("ID", uint(request.ID)), zap.Error(err))

		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}

	// Создаем объект типа Person на основе запроса
	person := models.Person{
		ID:        uint(request.ID),
		Name:      request.Name,
		Surname:   request.Surname,
		Age:       request.Age,
		Email:     request.Email,
		Telephone: request.Telephone,
	}

	// Обновляем информацию о человеке в базе данных
	err = h.Storage.PersonRepository.UpdatePerson(&person)
	if err != nil {
		// Проверяем, существует ли запись с данным Email кроме обновляемой
		if errors.Is(err, database.ErrEmailExists) {
			logging.Logger.Info("Email exists", zap.String("Email:", request.Email), zap.Error(err))
			fault := createSOAPFault("soap:Client", models.ErrorRecordEmailExistsMessage, models.ErrorRecordEmailExistsCode, models.ErrorRecordEmailExistsDetail)
			fmt.Printf("Response Fault: %+v\n", fault)
			c.XML(http.StatusConflict, fault)
			return
		}
		logging.Logger.Error("Error updating person with ID", zap.Uint("ID", uint(request.ID)), zap.Error(err))

		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}
	fmt.Println("Person updated with ID:")
	logging.Logger.Info("Successfully updated person with ID", zap.Uint("ID", uint(request.ID)))

	response := models.UpdatePersonResponse{
		Status: true,
	}

	// Возвращаем результат в формате XML
	fmt.Printf("Response: %+v\n", response)
	c.XML(http.StatusOK, response)
}

func (h *StorageHandler) getPersonHandler(c *gin.Context, request *models.GetPersonRequest) {
	// Получаем информацию о человеке по ID
	person, err := h.Storage.PersonRepository.GetPerson(request.ID)
	if err != nil {

		if errors.Is(err, database.ErrPersonNotFound) {
			fault := createSOAPFault("soap:Client", models.ErrorRecordNotFoundMessage, models.ErrorRecordNotFoundCode, models.ErrorRecordNotFoundDetail)
			fmt.Printf("Response Fault: %+v\n", fault)
			c.XML(http.StatusNotFound, fault)
			return
		}

		logging.Logger.Error("Error getting person with ID", zap.Uint("ID", uint(request.ID)), zap.Error(err))
		// Формируем SOAP Fault при ошибке
		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}

	// Если записи не найдены, формируем SOAP Fault для клиента
	if person == nil {
		fmt.Printf("No person found with ID %d\n", request.ID)

		fault := createSOAPFault("soap:Client", models.ErrorRecordNotFoundMessage, models.ErrorRecordNotFoundCode, models.ErrorRecordNotFoundDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusNotFound, fault)
		return
	}

	// Если человек найден, формируем ответ
	response := models.GetPersonResponse{
		Person: *person,
	}

	// Возвращаем результат в формате XML
	fmt.Printf("Response: %+v\n", response)
	c.XML(http.StatusOK, response)
}

// Метод получения всех записей
func (h *StorageHandler) getAllPersonsHandler(c *gin.Context) {
	// Получаем все записи из базы
	persons, err := h.Storage.PersonRepository.GetAllPersons()
	if err != nil {
		logging.Logger.Error("Error getting all persons", zap.Error(err))

		// Формируем SOAP Fault для ошибки получения
		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}

	// Если записи не найдены, формируем SOAP Fault для клиента
	if len(persons) == 0 {
		fmt.Println("No persons found.")

		fault := createSOAPFault("soap:Client", models.ErrorRecordNotFoundMessage, models.ErrorRecordNotFoundCode, models.ErrorRecordNotFoundDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusNotFound, fault)
		return
	}

	response := models.GetAllPersonsResponse{
		Persons: persons,
	}

	// Возвращаем результат в формате XML
	fmt.Printf("Response: %+v\n", response)
	c.XML(http.StatusOK, response)
}

// Метод удаления записи по ID
func (h *StorageHandler) deletePersonHandler(c *gin.Context, request *models.DeletePersonRequest) {
	//////////////////////////////////////
	if !h.BasicAuth(c) {
		logging.Logger.Error("Error Invalid user login or password")
		fault := createSOAPFault("soap:Client", models.ErrorAuthIncorrectMessage, models.ErrorAuthIncorrectCode, models.ErrorAuthIncorrectDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusUnauthorized, fault)
		return
	}
	//////////////////////////////////////
	//Проверяем существование записи по ID, если нет, формируем SOAP Fault
	checkByID, err := h.Storage.PersonRepository.CheckPersonByID(uint(request.ID))
	if !checkByID {
		logging.Logger.Error("Error getting person with ID", zap.Uint("ID", uint(request.ID)), zap.Error(err))
		fault := createSOAPFault("soap:Client", models.ErrorRecordNotFoundMessage, models.ErrorRecordNotFoundCode, models.ErrorRecordNotFoundDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusNotFound, fault)
		return
	}
	if err != nil {
		logging.Logger.Error("Error getting person with ID", zap.Uint("ID", uint(request.ID)), zap.Error(err))

		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}

	//Удаляем запись по ID из базы
	err = h.Storage.PersonRepository.DeletePerson(request)
	if err != nil {
		logging.Logger.Error("Error deleting person with ID", zap.Uint("ID", uint(request.ID)), zap.Error(err))

		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}

	logging.Logger.Info("Successfully deleted person with ID", zap.Uint("ID", uint(request.ID)))
	//Формируем статус в формате SOAP

	response := models.UpdatePersonResponse{
		Status: true,
	}
	fmt.Printf("Response: %+v\n", response)
	c.XML(http.StatusOK, response)

}

// Метод поиска записей по запросу
func (h *StorageHandler) searchPersonHandler(c *gin.Context, request *models.SearchPersonRequest) {

	persons, err := h.Storage.PersonRepository.SearchPerson(request.Query)
	if err != nil {
		logging.Logger.Error("Error searching for persons with query", zap.String("query", request.Query), zap.Error(err))

		fault := createSOAPFault("soap:Server", "Internal Server Error", "500", "An unexpected error occurred.")
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusInternalServerError, fault)
		return
	}

	if len(persons) == 0 {
		fmt.Println("No persons found.")

		fault := createSOAPFault("soap:Client", models.ErrorRecordNotFoundMessage, models.ErrorRecordNotFoundCode, models.ErrorRecordNotFoundDetail)
		fmt.Printf("Response Fault: %+v\n", fault)
		c.XML(http.StatusNotFound, fault)
		return
	} else {
		fmt.Printf("Found persons: %+v\n", persons)
	}

	// Формируем результат в формате SOAP
	response := models.SearchPersonResponse{
		Persons: persons,
	}
	fmt.Printf("Response: %+v\n", response)
	c.XML(http.StatusOK, response)
}
