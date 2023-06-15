package decrypt

import (
	"encoding/base64"
	"testing"
)

// Fake eyeball test where data is from NodeJS project
func TestEncryptedDataFromGameServerFake(t *testing.T) {
    sharedSecret, err := base64.StdEncoding.DecodeString("wN9DwvKgCcN3SAH2uzhS+A==")
    if err != nil {
        t.Fatal(err)
    }
    b64FromGameServer, err := base64.StdEncoding.DecodeString("e9ly/+hK+FKlKamKwnM8OxsQgNGGyf+Ecgo6vN3hRH/raBTObgLAWPjv7I8MLAFnIRtNDNbHSazYv73KlqZ73BGBcj3YUJQCd7Wni7QGPZlwVl6F15iP4lJecYCkVSwv5PXHQ0m3FNIMBjQOY3aPn6mecF1adoq9H0knpIPlcodYcjH5IVc6MHaoWLBLI0wiK3MAyqWqLL06hAIcRLlknoBPRSLP079NmjpZvc+t7vrxbRl8PdYMGcV8GfOTVJ3iw/2Yj/PvHoODp8gR4ear4N4ByJicWvMhjUyVYpnZJMlph/R1JNXn9gg41STdJupmd235OWv5J4sJ1eqFjEuTAImViBVUnrI1pfEins09kfF4a6fjP/UBWzwQ8Zzhp0bO0aWZxGSwYXEvCmX4thfLtVU8z4WAM1MdpHI9m1vizGf0wsSrzY7XH+6QFaEoz+lPkTdEwDsAwQW1xSCtHpWUy8lZLwFJ+aRr+gA5M5JyQAjCzGbNuRddW/0H5YEPcIeRks2l7/GvQclr+l0cV791+ABv5xsEgIG1ouPfBg==")
    if err != nil {
        t.Fatal(err)
    }

    decrypted, err := DecryptHandlePostedHiscores(sharedSecret, b64FromGameServer);
    if err != nil {
        t.Fatal(err)
    }

    t.Log(string(decrypted))
}

func TestMain(m *testing.M) {
    m.Run()
}
