package models

type GetAllPersonsResponse struct {
	Persons []Person `xml:"persons"`
}
type GetPersonResponse struct {
    Person Person `xml:"Person"` 
}


type ErrorResponse struct {
	Type     string `xml:"type"`
	Title    string `xml:"title"`
	Status   int    `xml:"status"`
	Detail   string `xml:"detail"`
	Instance string `xml:"instance"`
}

type DeleteResponse struct {
	Status bool `xml:"status"`
}

type SearchPersonResponse struct {
    Persons []Person `xml:"Persons"`
}


type AddPersonResponse struct {
    ID uint `xml:"ID"`
}

type UpdatePersonResponse struct {
    Status bool `xml:"status"`
}