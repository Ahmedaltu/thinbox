package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Run forks a child process inside new Linux namespaces.
// It re-execs the current binary with the "child" subcommand
// so the child starts fresh inside the new namespaces.
func Run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: run <image> <command>")
	}

	fmt.Printf("thinbox: starting container\n")
	fmt.Printf("thinbox: image=%s command=%s\n", args[0], args[1:])

	// Re-exec this binary as "child" inside new namespaces
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, args...)...)

	// Wire up stdin/stdout/stderr so the container is interactive
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Request new namespaces via clone flags
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container exited: %w", err)
	}

	return nil
}

// Child runs inside the new namespaces.
// It sets the hostname then execs the user command.
func Child(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("child: usage: <image> <command>")
	}

	command := args[1:]

	// Set container hostname
	if err := syscall.Sethostname([]byte("thinbox-container")); err != nil {
		return fmt.Errorf("sethostname: %w", err)
	}

	// Mount a fresh /proc so ps shows only container processes
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount /proc: %w", err)
	}
	defer syscall.Unmount("/proc", 0)

	fmt.Printf("thinbox: /proc mounted\n")

	// Find and exec the command
	path, err := exec.LookPath(command[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", command[0])
	}

	return syscall.Exec(path, command, os.Environ())
}
