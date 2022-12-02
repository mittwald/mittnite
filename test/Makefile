DOCKER_COMPOSE ?= docker-compose
STACK_FILE := docker-compose.integration.yml
STATE_FILE := state.txt

$(STATE_FILE):
	$(DOCKER_COMPOSE) -f $(STACK_FILE) up -d
	touch $@

.PHONY: clean
clean: env-down

.PHONY: env-up
env-up: $(STATE_FILE)

.PHONY: env-down
env-down:
	$(DOCKER_COMPOSE) -f $(STACK_FILE) down
	$(RM) $(STATE_FILE)
