package fast

import (
	"bytes"
	json "encoding/json"
	"fmt"
	"io"
	"strings"
)

type UserDefault struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

// using default json parser
func FastSearchDefault(out io.Writer, data []byte) {
	seenBrowsers := map[string]interface{}{}

	fmt.Fprintln(out, "found users:")

	user := UserDefault{}
	for i, line := range bytes.Split(data, []byte("\n")) {
		err := json.Unmarshal(line, &user)
		// err := user.UnmarshalJSON(line)
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
