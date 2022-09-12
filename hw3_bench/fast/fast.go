package fast

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type (
	//easyjson:json
	User struct {
		Browsers []string `json:"browsers,nocopy"`
		Company  string   `json:"company,nocopy"`
		Country  string   `json:"country,nocopy"`
		Email    string   `json:"email,nocopy"`
		Job      string   `json:"job,nocopy"`
		Name     string   `json:"name,nocopy"`
		Phone    string   `json:"phone,nocopy"`
	}
)

func FastSearch(out io.Writer, data []byte) {
	seenBrowsers := map[string]interface{}{}

	fmt.Fprintln(out, "found users:")

	user := User{}
	for i, line := range bytes.Split(data, []byte("\n")) {
		// err := json.Unmarshal(line, &user)
		err := user.UnmarshalJSON(line)
		if err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "Android") {
				seenBrowsers[browser] = nil
				isAndroid = true
				continue
			}

			if strings.Contains(browser, "MSIE") {
				seenBrowsers[browser] = nil
				isMSIE = true
				continue
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		mail := strings.Replace(user.Email, "@", " [at] ", 1)
		fmt.Fprintf(out, "[%d] %s <%s>\n", i, user.Name, mail)
	}

	fmt.Fprintln(out, "\nTotal unique browsers", len(seenBrowsers))
}
