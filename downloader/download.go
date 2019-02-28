package downloader

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/grafov/m3u8"
)

var keyCache map[string][]byte

func init() {
	keyCache = map[string][]byte{}
}

func Length(pls *m3u8.MediaPlaylist) int64 {
	l := int64(0)
	for _, s := range pls.Segments {
		if s != nil {
			l = l + s.Limit
		}
	}
	return l
}

func DownloadAfterBytes(w io.Writer, pls *m3u8.MediaPlaylist, n int64) error {
	l := int64(0)
	for i, s := range pls.Segments {
		if s != nil {
			l = l + s.Limit
			if l > n {
				return DownloadAfter(w, pls, i)
			}
		}
	}
	if n > l {
		return ErrOutOfRange
	}
	return nil
}

func Download(w io.Writer, pls *m3u8.MediaPlaylist) error {
	return DownloadAfter(w, pls, 0)
}

func DownloadAfter(w io.Writer, pls *m3u8.MediaPlaylist, i int) error {
	if i >= len(pls.Segments) || i < 0 {
		return ErrOutOfRange
	}

	type encData struct {
		data []byte
		key  *m3u8.Key
	}

	errCh := make(chan error)
	sigCh := make(chan *m3u8.MediaSegment, 1)
	encCh := make(chan *encData, 1)
	datCh := make(chan []byte, 1)
	ctx, done := context.WithCancel(context.Background())

	go func() {
		for s := range sigCh {
			data, err := FetchSigment(s)
			if err != nil {
				errCh <- err
				return
			}
			encCh <- &encData{data, s.Key}
		}
		close(encCh)
	}()

	go func() {
		for e := range encCh {
			data, err := DecodeSigment(e.data, e.key.URI, e.key.IV)
			if err != nil {
				errCh <- err
				return
			}
			datCh <- data
		}
		close(datCh)
	}()

	go func() {
		for d := range datCh {
			if _, err := w.Write(d); err != nil {
				errCh <- err
				return
			}
		}
		done()
	}()

	go func() {
		for i := i; i < len(pls.Segments); i++ {
			s := pls.Segments[i]
			if s != nil {
				sigCh <- s
			}
		}
		close(sigCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	return nil
}

func FetchSigment(s *m3u8.MediaSegment) ([]byte, error) {
	if strings.LastIndex(s.URI, "_") == -1 {
		return nil, ErrInvalidItemID
	}

	itemid := s.URI[:strings.LastIndex(s.URI, "_")]
	if len(itemid) < 3 {
		return nil, ErrInvalidItemID
	}

	u, _ := url.Parse("https://vod.bioscopelive.com/")
	u, _ = u.Parse(path.Join("/vod/vod", string(itemid[0]), string(itemid[1]), itemid, s.URI))

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://www.bioscopelive.com/")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Range", toByteRange(s.Offset, s.Limit))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, &HTTPError{res.StatusCode}
	}
	if res.ContentLength != s.Limit {
		return nil, &ContentLengthError{s.Limit, res.ContentLength}
	}

	return ioutil.ReadAll(res.Body)
}

func toByteRange(o, l int64) string {
	return fmt.Sprintf("bytes=%d-%d", o, (o + l - 1))
}

func DecodeSigment(src []byte, keyURI, keyIV string) ([]byte, error) {
	key, err := GetKey(keyURI)
	if err != nil {
		return nil, err
	}
	iv, err := PrepareIV(keyIV)
	if err != nil {
		return nil, err
	}

	dst := make([]byte, len(src))

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dst, src)

	return dst, nil
}

func PrepareIV(ivs string) ([]byte, error) {
	if strings.HasPrefix(ivs, "0x") || strings.HasPrefix(ivs, "0X") {
		ivs = ivs[2:]
	}
	iv, err := hex.DecodeString(ivs)
	if err != nil {
		return nil, err
	}
	return iv, nil
}

func GetKey(uri string) ([]byte, error) {
	key := keyCache[uri]
	if key != nil {
		return key, nil
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	u.Scheme = "http"

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://www.bioscopelive.com/")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	key, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	keyCache[uri] = key

	return key, nil
}
