package repository

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

const bareConfig = "[core]\n\trepositoryformatversion = 0\n\tfilemode = false\n\tbare = true\n\tlogallrefupdates = false\n"

// runGit is package-private so tests can exercise transport failures without
// credentials or a network server. Production always calls the installed Git.
var runGit = executeGit

func executeGit(ctx context.Context, environment []string, limit int64, arguments ...string) ([]byte, error) {
	program, err := exec.LookPath("git")
	if err != nil {
		return nil, failure(fault.Unsupported, "git_dependency")
	}
	if platform.ValidateExecutable(program) != nil {
		return nil, failure(fault.Trust, "git_dependency")
	}
	command := exec.CommandContext(ctx, program, arguments...)
	command.Env = environment
	command.Stderr = io.Discard
	output := &limitedOutput{remaining: limit}
	command.Stdout = output
	if err = command.Run(); err != nil {
		return nil, failure(fault.Unavailable, "git_command")
	}
	return output.buffer.Bytes(), nil
}

type limitedOutput struct {
	buffer    bytes.Buffer
	remaining int64
}

func (output *limitedOutput) Write(data []byte) (int, error) {
	if int64(len(data)) > output.remaining {
		return 0, failure(fault.Trust, "git_output_size")
	}
	output.remaining -= int64(len(data))
	return output.buffer.Write(data)
}

func (s *store) environment(authenticatedFetch, verifySignature bool) ([]string, error) {
	// Ambient Git, shell startup, and cloud credentials are deliberately absent.
	// These variables provide only executable discovery, OS support and TLS/proxy
	// transport. Trust in proxy/CA configuration belongs to the host administrator.
	allowed := map[string]bool{"PATH": true, "SystemRoot": true, "WINDIR": true, "TEMP": true, "TMP": true, "HTTP_PROXY": true, "HTTPS_PROXY": true, "ALL_PROXY": true, "NO_PROXY": true, "http_proxy": true, "https_proxy": true, "all_proxy": true, "no_proxy": true, "SSL_CERT_FILE": true, "SSL_CERT_DIR": true, "GIT_SSL_CAINFO": true}
	environment := make([]string, 0, 48)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if allowed[name] {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "GIT_NO_REPLACE_OBJECTS=1", "GIT_ATTR_NOSYSTEM=1", "GIT_OPTIONAL_LOCKS=0")
	settings := [][2]string{
		{"credential.helper", ""}, {"credential.interactive", "false"}, {"core.hooksPath", os.DevNull}, {"core.attributesFile", os.DevNull}, {"core.fsmonitor", "false"}, {"http.followRedirects", "false"}, {"http.sslVerify", "true"}, {"protocol.allow", "never"}, {"protocol.https.allow", "always"}, {"fetch.fsckObjects", "true"}, {"transfer.fsckObjects", "true"}, {"submodule.recurse", "false"}, {"fetch.writeCommitGraph", "false"}, {"maintenance.auto", "false"}, {"gc.auto", "0"},
	}
	if verifySignature {
		verifier, err := trustedSSHKeygen()
		if err != nil {
			return nil, err
		}
		settings = append(settings, [2]string{"gpg.format", "ssh"}, [2]string{"gpg.ssh.allowedSignersFile", s.repo.AllowedSignersFile}, [2]string{"gpg.ssh.program", verifier})
	}
	if authenticatedFetch && s.repo.TokenFile != "" {
		data, err := readProtected(s.repo.TokenFile, true, 16<<10)
		if err != nil {
			return nil, err
		}
		token, err := safeToken(data)
		if err != nil {
			return nil, err
		}
		// The token appears only in the child environment, never argv, local Git
		// configuration or errors. Other processes of the same OS identity remain
		// inside this trust boundary and may inspect the child's environment.
		settings = append(settings, [2]string{"http." + s.repo.URL + ".extraHeader", "Authorization: Bearer " + token})
	}
	for index, setting := range settings {
		environment = append(environment, "GIT_CONFIG_KEY_"+strconv.Itoa(index)+"="+setting[0], "GIT_CONFIG_VALUE_"+strconv.Itoa(index)+"="+setting[1])
	}
	environment = append(environment, "GIT_CONFIG_COUNT="+strconv.Itoa(len(settings)))
	return environment, nil
}

func (s *store) command(ctx context.Context, limit int64, arguments ...string) ([]byte, error) {
	return s.run(ctx, false, limit, arguments...)
}

func (s *store) verificationCommand(ctx context.Context, limit int64, arguments ...string) ([]byte, error) {
	return s.runVerified(ctx, false, limit, arguments...)
}

func (s *store) run(ctx context.Context, authenticatedFetch bool, limit int64, arguments ...string) ([]byte, error) {
	return s.runWithVerification(ctx, authenticatedFetch, false, limit, arguments...)
}

func (s *store) runVerified(ctx context.Context, authenticatedFetch bool, limit int64, arguments ...string) ([]byte, error) {
	return s.runWithVerification(ctx, authenticatedFetch, true, limit, arguments...)
}

func (s *store) runWithVerification(ctx context.Context, authenticatedFetch, verifySignature bool, limit int64, arguments ...string) ([]byte, error) {
	environment, err := s.environment(authenticatedFetch, verifySignature)
	if err != nil {
		return nil, err
	}
	return runGit(ctx, environment, limit, append([]string{"--git-dir=" + s.git}, arguments...)...)
}

// trustedSSHKeygen resolves the verifier before Git uses it. A PATH entry can
// be operator controlled, but it must not replace a verified native helper
// with a writable executable during commit-signature verification.
func trustedSSHKeygen() (string, error) {
	program, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return "", failure(fault.Unsupported, "ssh_signature_dependency")
	}
	if err = platform.ValidateExecutable(program); err != nil {
		return "", failure(fault.Trust, "ssh_signature_dependency")
	}
	return program, nil
}

func (s *store) initialize(ctx context.Context) error {
	if _, err := os.Lstat(s.git); err == nil {
		return s.checkGitConfig()
	} else if !os.IsNotExist(err) {
		return failure(fault.Trust, "git_directory")
	}
	emptyTemplate, err := os.MkdirTemp(s.repo.Directory, ".template-*")
	if err != nil {
		return failure(fault.Trust, "git_template")
	}
	defer os.RemoveAll(emptyTemplate)
	if _, err = s.command(ctx, 4096, "init", "--bare", "--quiet", "--template="+emptyTemplate, s.git); err != nil {
		return err
	}
	if err = platform.EnsurePrivateDirectory(s.git); err != nil {
		return failure(fault.Trust, "git_directory")
	}
	if err = platform.WriteProtected(filepath.Join(s.git, "config"), []byte(bareConfig)); err != nil {
		return failure(fault.Trust, "git_config")
	}
	return s.checkGitConfig()
}

func (s *store) checkGitConfig() error {
	if err := protectedDirectory(s.git); err != nil {
		return err
	}
	data, err := readProtected(filepath.Join(s.git, "config"), true, 4096)
	if err != nil || string(data) != bareConfig {
		return failure(fault.Trust, "git_config")
	}
	return nil
}

func (s *store) verifyCommit(ctx context.Context, commit string) error {
	data, err := s.command(ctx, 256<<10, "cat-file", "commit", commit)
	if err != nil {
		return failure(fault.Trust, "commit_object")
	}
	header, _, found := bytes.Cut(data, []byte("\n\n"))
	if !found || !bytes.Contains(header, []byte("\ngpgsig -----BEGIN SSH SIGNATURE-----\n")) {
		return failure(fault.Trust, "signature_format")
	}
	// Native Git/ssh-keygen verify the signed bytes against locally pinned
	// allowed-signers. Repository content cannot install keys or change policy.
	if _, err = s.verificationCommand(ctx, 4096, "verify-commit", commit); err != nil {
		return failure(fault.Trust, "signature")
	}
	return nil
}

func (s *store) ancestor(ctx context.Context, ancestor, commit string) error {
	if _, err := s.command(ctx, 4096, "merge-base", "--is-ancestor", ancestor, commit); err != nil {
		return failure(fault.Trust, "rollback_or_divergence")
	}
	return nil
}
