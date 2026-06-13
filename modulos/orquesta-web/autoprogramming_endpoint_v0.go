package orquestaweb

import "net/http"

const WebAutoprogrammingPageEndpointV0 = "/autoprogramming"

type AutoprogrammingWebEndpointV0 struct{}

func NewAutoprogrammingWebEndpointV0() AutoprogrammingWebEndpointV0 {
	return AutoprogrammingWebEndpointV0{}
}

func (endpoint AutoprogrammingWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handleWebPublicHTTPOptionsV0(w, r, http.MethodGet) {
		return
	}
	if r.Method != http.MethodGet {
		setWebPublicHTTPAllowV0(w, http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	writeWebHTMLStringResponseV0(w, http.StatusOK, autoprogrammingHTMLV0(), "es")
}
