# Centurion Build System

config:
```
type EnvVars struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Secrets struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type BuildConfig struct {
	Language string    `yaml:"language"`
	Version  string    `yaml:"version"`
	Cmd      string    `yaml:"cmd"`
	EnvVars  []EnvVars `yaml:"env_vars"`
	Flags    struct {
		Ldflags []string `yaml:"ldflags"`
	} `yaml:"flags"`
}

type RuntimeConfig struct {
	Service string `yaml:"service"`
}

type ServiceInfo struct {
	Name         string         `yaml:"name"`
	Template     string         `yaml:"template"`
	Path         string         `yaml:"-"`
	Config       map[string]any `yaml:"-"`
	BuildConfig  BuildConfig    `yaml:"build"`
	Runtime      RuntimeConfig  `yaml:"runtime"`
	EnvVars      []EnvVars      `yaml:"env_vars"`
	Secrets      []Secrets      `yaml:"secrets"`
	Region       string         `yaml:"region"`
	Project      string         `yaml:"project"`
	Provider     string         `yaml:"provider"`
	RegistryName string         `yaml:"registry_name"`
}
```


### Notes

In order to execute the commands I need some sort of wrapper it needs to do the following:
- load all the recepies with steps in order
- create and load the command registry
- search for the the service.yamls
- create a recepie for each service type we have found
- generate the commands for each step
-

Determine what to build notes:
- should I determine what services to build or leave that in the eye of the beholder (whoever attatches this to their repo?)
- I would like to smartly define a config for this and say here is the location of the shared folder, if we update that then re-build all services, otherwise only re-build if we touch repos
- is there even a way to dynamically go through a codebase and create a topological map of dependencies such that I can determine if a file in one folder got update that it affects another?
- do I even need that logic?
