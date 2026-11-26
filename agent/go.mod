module github.com/ekkywi/sailguard/agent

go 1.27.0

require (
	github.com/ekkywi/sailguard/pkgs v0.0.0
	github.com/go-ole/go-ole v1.3.0
	github.com/google/uuid v1.6.0
	golang.org/x/sys v0.49.0
)

replace github.com/ekkywi/sailguard/pkgs => ../pkgs
