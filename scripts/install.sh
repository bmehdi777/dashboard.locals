#!/bin/sh

set -eu

usage() {
	cat >&2 <<'EOF'
Usage: install.sh \
  --server-bin PATH \
  --cli-bin PATH \
  --bindir PATH \
  --systemd-user-dir PATH \
  --service-name NAME \
  --service-template PATH \
  [--install COMMAND] \
  [--systemctl COMMAND]
EOF
}

die() {
	printf 'dashboard.locals: %s\n' "$1" >&2
	exit 1
}

server_bin=
cli_bin=
bindir=
systemd_user_dir=
service_name=
service_template=
install_command=install
systemctl_command=systemctl

while [ "$#" -gt 0 ]; do
	case "$1" in
		--server-bin)
			[ "$#" -ge 2 ] || die "missing value for --server-bin"
			server_bin=$2
			shift 2
			;;
		--cli-bin)
			[ "$#" -ge 2 ] || die "missing value for --cli-bin"
			cli_bin=$2
			shift 2
			;;
		--bindir)
			[ "$#" -ge 2 ] || die "missing value for --bindir"
			bindir=$2
			shift 2
			;;
		--systemd-user-dir)
			[ "$#" -ge 2 ] || die "missing value for --systemd-user-dir"
			systemd_user_dir=$2
			shift 2
			;;
		--service-name)
			[ "$#" -ge 2 ] || die "missing value for --service-name"
			service_name=$2
			shift 2
			;;
		--service-template)
			[ "$#" -ge 2 ] || die "missing value for --service-template"
			service_template=$2
			shift 2
			;;
		--install)
			[ "$#" -ge 2 ] || die "missing value for --install"
			install_command=$2
			shift 2
			;;
		--systemctl)
			[ "$#" -ge 2 ] || die "missing value for --systemctl"
			systemctl_command=$2
			shift 2
			;;
		--help|-h)
			usage >&1
			exit 0
			;;
		*)
			usage
			die "unknown argument: $1"
			;;
	esac
done

[ -n "$server_bin" ] || die "--server-bin is required"
[ -n "$cli_bin" ] || die "--cli-bin is required"
[ -n "$bindir" ] || die "--bindir is required"
[ -n "$systemd_user_dir" ] || die "--systemd-user-dir is required"
[ -n "$service_name" ] || die "--service-name is required"
[ -n "$service_template" ] || die "--service-template is required"

[ -f "$server_bin" ] || die "server binary does not exist: $server_bin"
[ -f "$cli_bin" ] || die "CLI binary does not exist: $cli_bin"
[ -r "$service_template" ] || die "service template is not readable: $service_template"

case "$bindir" in
	/*) ;;
	*) die "--bindir must be an absolute path: $bindir" ;;
esac
case "$systemd_user_dir" in
	/*) ;;
	*) die "--systemd-user-dir must be an absolute path: $systemd_user_dir" ;;
esac
case "$service_name" in
	*/*) die "--service-name must not contain a slash: $service_name" ;;
esac

grep -q '^ExecStart=@EXEC_START@$' "$service_template" || \
	die "service template must contain an ExecStart=@EXEC_START@ entry"

mkdir -p "$bindir" "$systemd_user_dir"

server_target=$bindir/dashboard-server
cli_target=$bindir/dashboard
service_target=$systemd_user_dir/$service_name

server_tmp=
cli_tmp=
service_tmp=

cleanup() {
	[ -z "${server_tmp:-}" ] || rm -f "$server_tmp"
	[ -z "${cli_tmp:-}" ] || rm -f "$cli_tmp"
	[ -z "${service_tmp:-}" ] || rm -f "$service_tmp"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

server_tmp=$(mktemp "$bindir/.dashboard-server.XXXXXX")
cli_tmp=$(mktemp "$bindir/.dashboard.XXXXXX")
service_tmp=$(mktemp "$systemd_user_dir/.$service_name.XXXXXX")

# Copy to temporary files and rename them only after the copy is complete, so
# a concurrent CLI invocation cannot observe a partially installed binary.
"$install_command" -m 0755 "$server_bin" "$server_tmp"
mv -f "$server_tmp" "$server_target"
server_tmp=

"$install_command" -m 0755 "$cli_bin" "$cli_tmp"
mv -f "$cli_tmp" "$cli_target"
cli_tmp=

# The template has a single placeholder on the ExecStart line. Rendering it
# line by line avoids shell or sed expansion of paths containing special
# characters.
while IFS= read -r line || [ -n "$line" ]; do
	if [ "$line" = 'ExecStart=@EXEC_START@' ]; then
		printf 'ExecStart=%s\n' "$server_target"
	else
		printf '%s\n' "$line"
	fi
done < "$service_template" > "$service_tmp"
chmod 0644 "$service_tmp"
mv -f "$service_tmp" "$service_target"
service_tmp=

# Restart is intentional: unlike `enable --now`, it also activates a running
# service with the newly installed executable during an update.
"$systemctl_command" --user daemon-reload
"$systemctl_command" --user enable "$service_name"
"$systemctl_command" --user restart "$service_name"

trap - EXIT
trap - HUP INT TERM
printf 'Installed dashboard and dashboard-server in %s\n' "$bindir"
printf 'Enabled and restarted user service %s\n' "$service_name"
