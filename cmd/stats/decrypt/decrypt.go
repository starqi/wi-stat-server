package decrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

func DecryptHandlePostedHiscores(sharedSecret []byte, rawData []byte) ([]byte, error) {
    if sharedSecret == nil {
        return nil, errors.New("Cannot decrypt since no shared key is set up")
    } else {
        block, err := aes.NewCipher(sharedSecret)
        if err != nil {
            return nil, err
        }

        // Reminder: Must do length checks to avoid panics

        lenRawData := len(rawData)
        if lenRawData < 29 {
            return nil, errors.New("Too few bytes, rejecting")
        }
        iv := rawData[:12]
        ctWithTag := rawData[12:]

        gcm, err := cipher.NewGCM(block)
        if err != nil {
            return nil, err
        }
        p, err := gcm.Open(nil, iv, ctWithTag, nil)
        if err != nil {
            return nil, err
        }

        return p, nil
    }
}
