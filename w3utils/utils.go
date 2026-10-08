package w3utils

import "net/http"

func IsStateModifyingMethod(method string) bool {
	switch method {
	case http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete:
		return true
	default:
		return false
	}
}

func GetName(handler any) string {
	return "TODO"
}
