SHELL := /bin/bash

.PHONY: all
all: docker

.PHONY: docker
docker:
	docker build -t enrico204/http-telescope:latest .

.PHONY: push
push:
	docker push enrico204/http-telescope:latest

.PHONY: test
test:
	python3 -c "import yaml,subprocess; v=yaml.safe_load(open('.gitlab-ci.yml', 'r').read()); [subprocess.call(['/bin/bash', '-c', x]) for x in (v['code_check']['script'])]"
