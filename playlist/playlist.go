package playlist

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/grafov/m3u8"
)

func FetchMasterPlaylist(itemid string) (*m3u8.MasterPlaylist, error) {
	if len(itemid) < 3 {
		return nil, ErrInvalidItemID
	}

	u, _ := url.Parse("https://vod.bioscopelive.com/")
	u, _ = u.Parse(path.Join("/vod/vod", string(itemid[0]), string(itemid[1]), itemid, itemid+".m3u8"))

	res, err := doReq(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, &HTTPError{res.StatusCode}
	}

	plst, typ, err := m3u8.DecodeFrom(res.Body, true)
	if err != nil {
		return nil, err
	}
	if typ != m3u8.MASTER {
		return nil, ErrUnsupported
	}
	if plst == nil {
		return nil, ErrNotFound
	}
	return plst.(*m3u8.MasterPlaylist), nil
}

func FetchMediaPlaylist(uri string) (*m3u8.MediaPlaylist, error) {
	if strings.LastIndex(uri, "_") == -1 {
		return nil, ErrInvalidItemID
	}

	itemid := uri[:strings.LastIndex(uri, "_")]
	if len(itemid) < 3 {
		return nil, ErrInvalidItemID
	}

	u, _ := url.Parse("https://vod.bioscopelive.com/")
	u, _ = u.Parse(path.Join("/vod/vod", string(itemid[0]), string(itemid[1]), itemid, uri))

	res, err := doReq(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, &HTTPError{res.StatusCode}
	}

	plst, typ, err := m3u8.DecodeFrom(res.Body, true)
	if err != nil {
		return nil, err
	}
	if typ != m3u8.MEDIA {
		return nil, ErrUnsupported
	}
	if plst == nil {
		return nil, ErrNotFound
	}
	return plst.(*m3u8.MediaPlaylist), nil
}

func doReq(u *url.URL) (*http.Response, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://www.bioscopelive.com/")
	req.Header.Set("Connection", "keep-alive")

	return http.DefaultClient.Do(req)
}
