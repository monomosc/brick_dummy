package wfe_test

import (
	"brick/brickweb/wfe"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

var flattenedNewAccountJWS = []byte(`{
    "protected": "eyJub25jZSI6ICJwYksyVmo4TmI5eW5wSm1WbzVUb1JRIiwgInVybCI6ICJodHRwczovL2lkdmFjbWV0MDEuYmsuZGF0ZXYuZGUvbmV3LWFjY3QiLCAiYWxnIjogIlJTMjU2IiwgImp3ayI6IHsiZSI6ICJBUUFCIiwgImt0eSI6ICJSU0EiLCAibiI6ICJiUndSOGdUbXpwQVVlYklqQmZZcFFLdVpFTklzQVdRd2U5dmU4RkVDNE5MWWllMlktWUlPZjNtWFY1VzlGbm5XTkdTUDZnSy02VzJiU2I4UzdfZHZFYnhnT3JEbTBCOHQ0RE1KdGpBYlI0Tnl1b2pfdHVOdFU2cFk5dWlJOVJSWDRsb0cxamRIRWUtOFdBTW4xNmhVUWtYQW9jbmp5R1RRQU9kNkJkNExpTGIxZVd6bUl5VzVCbHhaeWxsbm1KRUlfbXRJTDU4TFduc3NLYTRmUDJWTGluSlEyZk5pa1R6MEcwelpUTGNobHdKblFud3RETjBGZ3NxaDNLcFlwVlZzSlRnbjh3emhfanFhUXVMd0IyYnVpQ2FKNlNUNmVNbXl4QjlnUkwyNHZBU3dWYmZ6WDRzM3BCQWpOblo2UE52WFBoZ09kZUNtQk92VDNuZUZlNEVYMlEifX0",
    "payload": "eyJ0ZXJtc09mU2VydmljZUFncmVlZCI6IHRydWV9",
    "signature": "WG1DppuKruZ5R5DWobOjrCDD73RwKbWUe5Cf_YZogMvVi_ICf0vDWTcQCr4RHk4BFEL3PIkpHULBDPSHEeJWvXFuVDmeu_07YDc4JKmTvSxFiYrb0bOYcB7y7_it7UR8wtiNlxRgdbysXjQjb7cz3VzVEIsAVCs93sWvtQYkMDLvfQNCLUtmcugDioJIl5Ql5aRDd3IqeUcN4L7n7_rS0tPh9SyppxjEBCMbKQdtncJNdR_SjFggPdD0uVRpMAP6QUz2RYBIaijB-9GOONmD7uq2pF0jkJWztvidBrkseUyoRcEUBN52aWCNHUjKUsYDgHbEtTeSXBjZgBFAuyXiGA"
}`)
var flattenedNewOrderJWS = []byte(`{
	"protected": "eyJub25jZSI6ICJfckk1SEhhaHZwS3FwQzRtQllGSlhnIiwgInVybCI6ICJodHRwczovL2lkdmFjbWV0MDEuYmsuZGF0ZXYuZGUvbmV3LW9yZGVyIiwgImFsZyI6ICJSUzI1NiIsICJraWQiOiAiaHR0cHM6Ly9pZHZhY21ldDAxLmJrLmRhdGV2LmRlL2FjY3QvNDhhNDgwNjZmYmZhODAxYjNiMzA5N2IxNDU4NzZkNmZkN2VkYTFjMjE0NjEyOWM1NTIyYTBjZTQ4MTA1ZGM5YiJ9", 
	"payload": "eyJpZGVudGlmaWVycyI6IFt7InR5cGUiOiJkbnMiLCJ2YWx1ZSI6ImNkeHRlc3RyZXMwMi5iay5kYXRldi5kZSJ9XX0", 
	"signature": "GSn9qvtZfWeRUcKuyzCbGplUuFpDN_t9HzaAgrAoaTAZCfWo_ZAOvDgk9M6grlRZ9MjnyZJSoIkdm4DWUBUraPuCpHVcRKU50KYlFIY8d4kuumGgM4zZe8ybs68qErJvrt7atKGX5xDRjejexHo-oNXBNpaejzI1ADyotTqTLg47Xr_PEDbNBxWmM8jFYFZnrmabXgX0eM2XhJWky9d0H_MFM7RLmrFFAUZoIMuSQIMkUboMy3RnS2Sa19ZmSmm22jtAyJo9j74nG9oamN6JS94YwVpMreecvcNAIHgg3RwRJBsvX2-pCCw4GJFLNa9KXRHmIBQFAUvAsxf0NOlabg"
}`)

func getDevWFE() *wfe.WebFrontEndImpl {
	w := wfe.New(logrus.New(), &mockCA{}, &mockStorage{}, &mockVa{})
	w.BasePath = "https://idvacmet01.bk.datev.de"
	return w
}

func TestParseJwsFlattened(t *testing.T) {
	frontend := getDevWFE()
	_, err := frontend.ParseJWS(context.Background(), flattenedNewAccountJWS)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

}
