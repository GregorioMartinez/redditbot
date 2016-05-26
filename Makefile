NAME = redditbot

build:
	docker build --tag $(NAME) . 

# Creates a container
# Runs a command
create:
	docker run --interactive --tty --env-file reddit-wikipediaposter.env --entrypoint=/go/bin/redditbot redditbot
