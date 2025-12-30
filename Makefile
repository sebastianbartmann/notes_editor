.PHONY: run install status

run:
	uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000

install:
	sudo cp notes-editor.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable notes-editor
	sudo systemctl restart notes-editor

status:
	sudo systemctl status notes-editor
