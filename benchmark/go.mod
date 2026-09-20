module github.com/pwn-model/pwn/benchmark

go 1.27.1

replace github.com/pwn-model/pwn v0.0.0 => ..

require (
	github.com/mlange-42/ark v0.8.3
	github.com/mlange-42/ark-tools v0.3.1
	github.com/pwn-model/pwn v0.0.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
