package core

import "github.com/matcornic/hermes"

type MailTheme interface {
	hermes.Theme
	GetStyle() string
}
