package filters

type Filter interface {
	Process(content string) string
	Name() string
}

var Registry = make(map[string]Filter)

func Register(filter Filter) {
	Registry[filter.Name()] = filter
}

func Get(name string) Filter {
	return Registry[name]
}
