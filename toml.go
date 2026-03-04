package sex

import "github.com/pelletier/go-toml/v2"

func UnmarshalTOML(data []byte, v any) error {
	return toml.Unmarshal(data, v)
}
