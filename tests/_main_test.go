package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestBitbucketProvider(t *testing.T) {
	// Leemos el fichero con el json
	file, err := os.Open("../tests/webhooks/bitbucket/pr_comment_created.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Leer el contenido del archivo
	jsonContent, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	// Creamos una solicitud de prueba falsa
	req, err := http.NewRequest("POST", "/api/v1/webhook/bitbucket", bytes.NewBuffer(jsonContent))
	if err != nil {
		t.Fatal(err)
	}

	// Creamos un ResponseRecorder (implementa ResponseWriter) para grabar la respuesta
	rr := httptest.NewRecorder()

	// Creamos un handler HTTP falso utilizando nuestro handler real
	router := http.NewServeMux()
	handleProvider("POST /api/v1/webhook/bitbucket", router, BitbucketProvider{})
	handler, _ := router.Handler(req)
	//handler := http.HandlerFunc()

	// Llamamos al método ServeHTTP del handler con nuestra solicitud falsa y nuestro ResponseRecorder
	handler.ServeHTTP(rr, req)

	// Verificamos el código de estado de la respuesta
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verificamos el cuerpo de la respuesta
	//expected := "Hello, world!"
	//if rr.Body.String() != expected {
	//	t.Errorf("handler returned unexpected body: got %v want %v",
	//		rr.Body.String(), expected)
	//}
}
