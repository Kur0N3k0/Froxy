package proxy

import "github.com/valyala/fasthttp"

var MrRules []RegexMatchReplace = make([]RegexMatchReplace, 0)

func AddReplaceRule(rule *RegexMatchReplace) {
	MrRules = append(MrRules, *rule)
}

func ReplaceMatchedString(matched string, mType string) string {
	for _, rule := range MrRules {
		if rule.Type == mType && rule.Enabled {
			matched = rule.Regex.ReplaceAllLiteralString(matched, rule.Replace)
		}
	}
	return matched
}

func ReplaceMatchedBytes(matched []byte, mType string) []byte {
	return []byte(ReplaceMatchedString(string(matched), mType))
}

func ReplaceMatchedRequest(req *fasthttp.Request) {
	req.Header.VisitAll(func(key, value []byte) {
		if string(key) == "Cookie" {
			return
		}
		req.Header.SetBytesKV(key, ReplaceMatchedBytes(value, MR_REQUEST_HEADER))
	})

	req.Header.VisitAllCookie(func(key, value []byte) {
		req.Header.SetCookieBytesKV(key, ReplaceMatchedBytes(value, MR_REQUEST_COOKIE))
	})

	req.SetBody(ReplaceMatchedBytes(req.Body(), MR_REQUEST_BODY))
}

func ReplaceMatchedResponse(resp *fasthttp.Response) {
	resp.Header.VisitAll(func(key, value []byte) {
		resp.Header.SetBytesKV(key, ReplaceMatchedBytes(value, MR_RESPONSE_HEADER))
	})
	resp.SetBody(ReplaceMatchedBytes(resp.Body(), MR_RESPONSE_BODY))
}
