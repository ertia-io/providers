package k3s

import (
	_ "embed"
)

//go:embed install_k3s.sh
var installer string
