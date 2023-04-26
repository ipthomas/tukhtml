package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/ipthomas/tukcnst"
	"github.com/ipthomas/tukdbint"
	"github.com/ipthomas/tukutil"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var htmlTemplates *template.Template
var data = make(map[string]interface{})

func main() {
	lambda.Start(Handle_Request)
}
func Handle_Request(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	log.SetFlags(log.Lshortfile)

	var err error
	dbconn := tukdbint.TukDBConnection{DBUser: os.Getenv(tukcnst.ENV_DB_USER), DBPassword: os.Getenv(tukcnst.ENV_DB_PASSWORD), DBHost: os.Getenv(tukcnst.ENV_DB_HOST), DBPort: os.Getenv(tukcnst.ENV_DB_PORT), DBName: os.Getenv(tukcnst.ENV_DB_NAME)}
	if err = tukdbint.NewDBEvent(&dbconn); err != nil {
		return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
	}
	if err = cacheTemplates(); err != nil {
		return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
	}

	log.Printf("Processing API Gateway %s Request Path %s", req.HTTPMethod, req.Path)
	var tplReturn bytes.Buffer
	body := []byte(req.Body)
	log.Println("Request Body")
	log.Println(string(body))
	switch req.QueryStringParameters[tukcnst.ACT] {
	case tukcnst.SELECT:
		if err = json.Unmarshal(body, &data); err != nil {
			log.Println(err.Error())
			return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
		}
		if err := htmlTemplates.ExecuteTemplate(&tplReturn, req.QueryStringParameters["name"], data); err != nil {
			log.Println(err.Error())
			return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
		}
		log.Println("Returning Response")
		log.Println(tplReturn.String())
		return queryResponse(http.StatusOK, tplReturn.String(), tukcnst.TEXT_HTML)
	case tukcnst.INSERT:
		tmplts := tukdbint.Templates{Action: tukcnst.DELETE}
		tmplt := tukdbint.Template{Name: req.QueryStringParameters["name"]}
		tmplts.Templates = append(tmplts.Templates, tmplt)
		tukdbint.NewDBEvent(&tmplts)
		tmplts = tukdbint.Templates{Action: tukcnst.INSERT}
		tmplt = tukdbint.Template{Name: req.QueryStringParameters["name"], IsXML: false, Template: req.Body}
		tmplts.Templates = append(tmplts.Templates, tmplt)
		if err = tukdbint.NewDBEvent(&tmplts); err != nil {
			return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
		}
	}
	return queryResponse(http.StatusOK, string(body), tukcnst.APPLICATION_JSON)
}

func cacheTemplates() error {
	var err error
	htmlTemplates = template.New(tukcnst.HTML)
	tmplts := tukdbint.Templates{Action: tukcnst.SELECT}
	tukdbint.NewDBEvent(&tmplts)
	log.Printf("cached %v Templates", tmplts.Count)
	funcmap := getTemplateFuncMap()
	for _, tmplt := range tmplts.Templates {
		if htmlTemplates, err = htmlTemplates.New(tmplt.Name).Funcs(funcmap).Parse(tmplt.Template); err != nil {
			log.Println(err.Error())
			return err
		}
	}
	return err
}
func getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"newuuid":          tukutil.NewUuid,
		"newid":            tukutil.Newid,
		"newzulu":          tukutil.Newzulu,
		"new30mfuturezulu": tukutil.New30mfutureyearzulu,
		"newdatetime":      tukutil.Newdatetime,
		"splitfhiroid":     tukutil.SplitFhirOid,
		"splitexpression":  tukutil.SplitExpression,
		"geticon":          tukutil.GetGlypicon,
		"mappedid":         tukdbint.GetIDMapsMappedId,
		"prettytime":       tukutil.PrettyTime,
	}
}
func setAwsResponseHeaders(contentType string) map[string]string {
	awsHeaders := make(map[string]string)
	awsHeaders["Server"] = "TUK_Event_Consumer_Proxy"
	awsHeaders["Access-Control-Allow-Origin"] = "*"
	awsHeaders["Access-Control-Allow-Headers"] = "accept, Content-Type"
	awsHeaders["Access-Control-Allow-Methods"] = "GET, POST, OPTIONS"
	awsHeaders[tukcnst.CONTENT_TYPE] = contentType
	return awsHeaders
}
func queryResponse(statusCode int, body string, contentType string) (*events.APIGatewayProxyResponse, error) {
	return &events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    setAwsResponseHeaders(contentType),
		Body:       body,
	}, nil
}
