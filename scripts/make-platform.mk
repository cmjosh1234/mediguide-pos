# Use one predictable shell, including when make is launched from PowerShell.
empty :=
space := $(empty) $(empty)
ifeq ($(OS),Windows_NT)
SHELL := cmd.exe
.SHELLFLAGS := /d /s /c
PYTHON ?= python
EXEEXT := .exe
# Avoid accidentally selecting the WSL bash.exe from System32.
GIT_BASH ?= C:/Program Files/Git/bin/bash.exe
BASH := "$(GIT_BASH)"
# Keep container paths intact when Bash scripts invoke Docker Desktop.
export MSYS2_ARG_CONV_EXCL := /app/;$(MSYS2_ARG_CONV_EXCL)
else
SHELL := /bin/sh
PYTHON ?= python3
EXEEXT :=
BASH := bash
endif
