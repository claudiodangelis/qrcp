package application

type Flags struct {
	Quiet             bool
	KeepAlive         bool
	ListAllInterfaces bool
	Port              int
	Path              string
	Interface         string
	Bind              string
	FQDN              string
	Zip               bool
	Config            string
	Browser           bool
	Secure            bool
	TlsCert           string
	TlsKey            string
	Output            string
	Reversed          bool
}

type App struct {
	Flags Flags
	Name  string
}

func New() App {
	return App{
		Name:  "qrcp",
		Flags: Flags{},
	}
}
