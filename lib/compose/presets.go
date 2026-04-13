// Package compose generates docker-compose.yml files from discovered Pilum services
// and infrastructure dependencies declared in .pilum.yml.
package compose

import (
	"fmt"
	"sort"
)

// Preset defines the default configuration for a built-in infrastructure type.
type Preset struct {
	Image           string
	DefaultPort     int
	HealthCmd       func(Dependency) string
	ConnStringTmpl  func(Dependency) string
	DefaultUser     string
	DefaultPassword string
	DefaultDatabase string
	Env             func(Dependency) []string
	Volumes         func(Dependency) []string
}

var presets = map[string]Preset{
	"postgres": {
		Image:           "postgres:15-alpine",
		DefaultPort:     5432,
		DefaultUser:     "appuser",
		DefaultPassword: "apppassword",
		DefaultDatabase: "appdb",
		HealthCmd: func(d Dependency) string {
			return fmt.Sprintf("pg_isready -U %s -d %s", d.User, d.Database)
		},
		ConnStringTmpl: func(d Dependency) string {
			return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
				d.User, d.Password, d.Name, d.Port, d.Database)
		},
		Env: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("POSTGRES_DB=%s", d.Database),
				fmt.Sprintf("POSTGRES_USER=%s", d.User),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", d.Password),
				"PGDATA=/var/lib/postgresql/data/pgdata",
			}
		},
		Volumes: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("%s_data:/var/lib/postgresql/data", d.Name),
				"./init-scripts:/docker-entrypoint-initdb.d",
			}
		},
	},
	"mysql": {
		Image:           "mysql:8.0",
		DefaultPort:     3306,
		DefaultUser:     "appuser",
		DefaultPassword: "apppassword",
		DefaultDatabase: "appdb",
		HealthCmd: func(_ Dependency) string {
			return "mysqladmin ping -h localhost"
		},
		ConnStringTmpl: func(d Dependency) string {
			return fmt.Sprintf("mysql://%s:%s@%s:%d/%s",
				d.User, d.Password, d.Name, d.Port, d.Database)
		},
		Env: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("MYSQL_DATABASE=%s", d.Database),
				fmt.Sprintf("MYSQL_USER=%s", d.User),
				fmt.Sprintf("MYSQL_PASSWORD=%s", d.Password),
				fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", d.Password),
			}
		},
		Volumes: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("%s_data:/var/lib/mysql", d.Name),
			}
		},
	},
	"mariadb": {
		Image:           "mariadb:11",
		DefaultPort:     3306,
		DefaultUser:     "appuser",
		DefaultPassword: "apppassword",
		DefaultDatabase: "appdb",
		HealthCmd: func(_ Dependency) string {
			return "mariadb-admin ping -h localhost"
		},
		ConnStringTmpl: func(d Dependency) string {
			return fmt.Sprintf("mysql://%s:%s@%s:%d/%s",
				d.User, d.Password, d.Name, d.Port, d.Database)
		},
		Env: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("MARIADB_DATABASE=%s", d.Database),
				fmt.Sprintf("MARIADB_USER=%s", d.User),
				fmt.Sprintf("MARIADB_PASSWORD=%s", d.Password),
				fmt.Sprintf("MARIADB_ROOT_PASSWORD=%s", d.Password),
			}
		},
		Volumes: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("%s_data:/var/lib/mysql", d.Name),
			}
		},
	},
	"redis": {
		Image:       "redis:7-alpine",
		DefaultPort: 6379,
		HealthCmd: func(_ Dependency) string {
			return "redis-cli ping"
		},
		ConnStringTmpl: func(d Dependency) string {
			return fmt.Sprintf("redis://%s:%d", d.Name, d.Port)
		},
		Env: func(_ Dependency) []string {
			return nil
		},
		Volumes: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("%s_data:/data", d.Name),
			}
		},
	},
	"mongo": {
		Image:           "mongo:7",
		DefaultPort:     27017,
		DefaultUser:     "appuser",
		DefaultPassword: "apppassword",
		DefaultDatabase: "appdb",
		HealthCmd: func(_ Dependency) string {
			return `mongosh --eval "db.adminCommand('ping')"`
		},
		ConnStringTmpl: func(d Dependency) string {
			return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
				d.User, d.Password, d.Name, d.Port, d.Database)
		},
		Env: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("MONGO_INITDB_ROOT_USERNAME=%s", d.User),
				fmt.Sprintf("MONGO_INITDB_ROOT_PASSWORD=%s", d.Password),
				fmt.Sprintf("MONGO_INITDB_DATABASE=%s", d.Database),
			}
		},
		Volumes: func(d Dependency) []string {
			return []string{
				fmt.Sprintf("%s_data:/data/db", d.Name),
			}
		},
	},
}

// GetPreset returns the preset for the given type, or false if not a built-in type.
func GetPreset(typeName string) (Preset, bool) {
	p, ok := presets[typeName]
	return p, ok
}

// SupportedTypes returns the sorted list of built-in dependency types.
func SupportedTypes() []string {
	types := make([]string, 0, len(presets))
	for t := range presets {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}
