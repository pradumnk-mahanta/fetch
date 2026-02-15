// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    torBoxAPIRespose, err := UnmarshalTorBoxAPIRespose(bytes)
//    bytes, err = torBoxAPIRespose.Marshal()

package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"
)

func UnmarshalTorBoxAPIRespose(data []byte) (TorBoxAPIRespose, error) {
	var r TorBoxAPIRespose
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TorBoxAPIRespose) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type TorBoxAPIRespose struct {
	Success bool        `json:"success"`
	Error   interface{} `json:"error"`
	Detail  string      `json:"detail"`
	Data    *Data       `json:"data"`
}

type DAT struct {
	ID                *int64     `json:"id,omitempty"`
	CreatedAt         *time.Time `json:"created_at,omitempty"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
	AuthID            string     `json:"auth_id"`
	Name              *string    `json:"name,omitempty"`
	Hash              string     `json:"hash"`
	DownloadState     *string    `json:"download_state,omitempty"`
	DownloadSpeed     *int64     `json:"download_speed,omitempty"`
	OriginalURL       *string    `json:"original_url,omitempty"`
	Eta               *int64     `json:"eta,omitempty"`
	Progress          *float64   `json:"progress,omitempty"`
	Size              *int64     `json:"size,omitempty"`
	DownloadID        *string    `json:"download_id,omitempty"`
	Files             []File     `json:"files,omitempty"`
	Active            *bool      `json:"active,omitempty"`
	Cached            *bool      `json:"cached,omitempty"`
	DownloadPresent   *bool      `json:"download_present,omitempty"`
	DownloadFinished  *bool      `json:"download_finished,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	Server            *int64     `json:"server,omitempty"`
	CachedAt          *time.Time `json:"cached_at,omitempty"`
	AlternativeHashes []string   `json:"alternative_hashes,omitempty"`
	Magnet            *string    `json:"magnet,omitempty"`
	Seeds             *int64     `json:"seeds,omitempty"`
	Peers             *int64     `json:"peers,omitempty"`
	Ratio             *float64   `json:"ratio,omitempty"`
	UploadSpeed       *int64     `json:"upload_speed,omitempty"`
	TorrentFile       *bool      `json:"torrent_file,omitempty"`
	DownloadPath      *string    `json:"download_path,omitempty"`
	Availability      *int64     `json:"availability,omitempty"`
	Tracker           *string    `json:"tracker,omitempty"`
	TotalUploaded     *int64     `json:"total_uploaded,omitempty"`
	TotalDownloaded   *int64     `json:"total_downloaded,omitempty"`
	Owner             *string    `json:"owner,omitempty"`
	SeedTorrent       *bool      `json:"seed_torrent,omitempty"`
	AllowZipped       *bool      `json:"allow_zipped,omitempty"`
	LongTermSeeding   *bool      `json:"long_term_seeding,omitempty"`
	TrackerMessage    *string    `json:"tracker_message,omitempty"`
	Private           *bool      `json:"private,omitempty"`
	UsenetdownloadID  *int64     `json:"usenetdownload_id,omitempty"`
	TorrentID         *int64     `json:"torrent_id,omitempty"`
}

type File struct {
	ID                int64  `json:"id"`
	Md5               string `json:"md5"`
	Hash              string `json:"hash"`
	Name              string `json:"name"`
	Size              int64  `json:"size"`
	Zipped            bool   `json:"zipped"`
	S3Path            string `json:"s3_path"`
	Infected          bool   `json:"infected"`
	Mimetype          string `json:"mimetype"`
	ShortName         string `json:"short_name"`
	AbsolutePath      string `json:"absolute_path"`
	OpensubtitlesHash string `json:"opensubtitles_hash"`
}

type Data struct {
	DAT      *DAT
	DATArray []DAT
	String   *string
}

func (x *Data) UnmarshalJSON(data []byte) error {
	x.DATArray = nil
	x.DAT = nil
	var c DAT
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, true, &x.DATArray, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.DAT = &c
	}
	return nil
}

func (x *Data) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, x.DATArray != nil, x.DATArray, x.DAT != nil, x.DAT, false, nil, false, nil, false)
}

func unmarshalUnion(data []byte, pi **int64, pf **float64, pb **bool, ps **string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) (bool, error) {
	if pi != nil {
		*pi = nil
	}
	if pf != nil {
		*pf = nil
	}
	if pb != nil {
		*pb = nil
	}
	if ps != nil {
		*ps = nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}

	switch v := tok.(type) {
	case json.Number:
		if pi != nil {
			i, err := v.Int64()
			if err == nil {
				*pi = &i
				return false, nil
			}
		}
		if pf != nil {
			f, err := v.Float64()
			if err == nil {
				*pf = &f
				return false, nil
			}
			return false, errors.New("Unparsable number")
		}
		return false, errors.New("Union does not contain number")
	case float64:
		return false, errors.New("Decoder should not return float64")
	case bool:
		if pb != nil {
			*pb = &v
			return false, nil
		}
		return false, errors.New("Union does not contain bool")
	case string:
		if haveEnum {
			return false, json.Unmarshal(data, pe)
		}
		if ps != nil {
			*ps = &v
			return false, nil
		}
		return false, errors.New("Union does not contain string")
	case nil:
		if nullable {
			return false, nil
		}
		return false, errors.New("Union does not contain null")
	case json.Delim:
		if v == '{' {
			if haveObject {
				return true, json.Unmarshal(data, pc)
			}
			if haveMap {
				return false, json.Unmarshal(data, pm)
			}
			return false, errors.New("Union does not contain object")
		}
		if v == '[' {
			if haveArray {
				return false, json.Unmarshal(data, pa)
			}
			return false, errors.New("Union does not contain array")
		}
		return false, errors.New("Cannot handle delimiter")
	}
	return false, errors.New("Cannot unmarshal union")
}

func marshalUnion(pi *int64, pf *float64, pb *bool, ps *string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) ([]byte, error) {
	if pi != nil {
		return json.Marshal(*pi)
	}
	if pf != nil {
		return json.Marshal(*pf)
	}
	if pb != nil {
		return json.Marshal(*pb)
	}
	if ps != nil {
		return json.Marshal(*ps)
	}
	if haveArray {
		return json.Marshal(pa)
	}
	if haveObject {
		return json.Marshal(pc)
	}
	if haveMap {
		return json.Marshal(pm)
	}
	if haveEnum {
		return json.Marshal(pe)
	}
	if nullable {
		return json.Marshal(nil)
	}
	return nil, errors.New("Union must not be null")
}

func (l DAT) ToJSON() (string, error) {
	bytes, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (d *DAT) LoadJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), d)
}
