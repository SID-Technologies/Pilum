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

Service Type Notes:
- should I add a service type, what if people have different kinds of services?
- then I should exeecute commands in a different order
- For each type I could have a command set list (aka a config for commands)


Docker Images:
- build executable
- create image
- push to location


Brew:
- build
- commit in assigned repo?
- publish merge cr?


GCP Cloud Run:
- build
- create images
- push to location (registry)
- publish to cloudrun


Service type logic:
- get type of each service
- create a command executor for that type 
- add commands to that type ie. dry run commands
- run each executor in order (GCP Cloud run executor, Brew executor)

I think I want the above logic unsure if I need the below logic:


Determine what to build notes:
- should I determine what services to build or leave that in the eye of the beholder (whoever attatches this to their repo?)
- I would like to smartly define a config for this and say here is the location of the shared folder, if we update that then re-build all services, otherwise only re-build if we touch repos
- is there even a way to dynamically go through a codebase and create a topological map of dependencies such that I can determine if a file in one folder got update that it affects another?
- do I even need that logic?