package storage

// DatasetDocument is the document to save to underlying storage.
// JSON looks like this:
//	{
//	    "id": <>,
//	    "downloads": {
//	        "csv": {
//	            "private": <>,
//	            "public": <>,
//              "size": 123
//	        },
//	        "csvw": {
//	            "private": <>,
//	            "public": <>,
//              "size": 456
//	        },
//	        "xls": {
//	            "private": <>,
//	            "public": <>,
//              "size": 789
//	        }
//      },
//      "links": {
//          "datasetversion": {
//              "href": "/datasets/id/editions/id/versions/4"
//          },
//          "self": {
//              "href": "/downloads/123"
//          }
//      }
//  }
//
type DatasetDocument struct {
	ID        string                             `json:"id"`
	Downloads map[string]DatasetDocumentDownload `json:"downloads"`
	Links     map[string]DatasetDocumentHref     `json:"links"`
}

type DatasetDocumentDownload struct {
	Private string `json:"private"`
	Public  string `json:"public"`
	Size    int    `json:"size"`
}

type DatasetDocumentHref struct {
	Href string `json:"href"`
}
