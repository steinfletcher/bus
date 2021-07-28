module github.com/steinfletcher/bus/example

go 1.16

replace github.com/steinfletcher/bus => ../

require (
	github.com/labstack/echo/v4 v4.4.0
	github.com/steinfletcher/apitest v1.5.11
	github.com/steinfletcher/bus v0.0.0-00010101000000-000000000000
)
