package x12

import "time"

type Document struct {
	Raw           string
	Engine        string
	ClaimID       string
	ConfigVersion int32
	GeneratedAt   time.Time
}

func NewPlaceholder(engine, claimID string, configVersion int32) Document {
	return Document{
		Raw:           engine + ":" + claimID + ":" + itoa(configVersion),
		Engine:        engine,
		ClaimID:       claimID,
		ConfigVersion: configVersion,
		GeneratedAt:   time.Now().UTC(),
	}
}

func itoa(v int32) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	n := int64(v)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
