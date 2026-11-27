build-%:
	docker build --build-arg DISTRO=$* -t container-image-fixture:$* .
	test "$$(docker image inspect -f '{{index .Config.Labels "distro"}}' container-image-fixture:$*)" = "$*"

push-%:
	@echo "push-$* ran on a run that must not push" >&2; exit 1

# + makes the recipe run even under make -n/-q, so the probe must not reach it
push-default:
	+@echo "push-default recipe ran" >&2; exit 1
