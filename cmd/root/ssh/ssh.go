//	Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// ssh package uses ssh and scp binaries to execute commands remotely and translate the results back to the calling node
package ssh

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/dremio/dremio-diagnostic-collector/cmd/root/cli"
	"github.com/dremio/dremio-diagnostic-collector/pkg/simplelog"
)

type Args struct {
	SSHKeyLoc      string
	SSHUser        string
	SudoUser       string
	ExecutorStr    string
	CoordinatorStr string
}

func NewCmdSSHActions(sshArgs Args) *CmdSSHActions {
	return &CmdSSHActions{
		cli:            &cli.Cli{},
		sshKey:         sshArgs.SSHKeyLoc,
		sshUser:        sshArgs.SSHUser,
		sudoUser:       sshArgs.SudoUser,
		executorStr:    sshArgs.ExecutorStr,
		coordinatorStr: sshArgs.CoordinatorStr,
	}
}

// CmdSSHActions depends on the scp and ssh programs being present and
// then assumes ssh public key auth is in place since it has no support for using
// password based authentication
type CmdSSHActions struct {
	cli            cli.CmdExecutor
	sshKey         string
	sshUser        string
	sudoUser       string
	executorStr    string
	coordinatorStr string
}

func (c *CmdSSHActions) Name() string {
	return "SSH/SCP"
}

func (c *CmdSSHActions) HostExecuteAndStream(mask bool, hostString string, output cli.OutputHandler, pat string, args ...string) (err error) {
	sshArgs := []string{"ssh", "-i", c.sshKey, "-o", "LogLevel=error", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no"}
	sshArgs = append(sshArgs, fmt.Sprintf("%v@%v", c.sshUser, hostString))
	sshArgs = c.addSSHUser(sshArgs)
	sshArgs = append(sshArgs, strings.Join(args, " "))
	return c.cli.ExecuteAndStreamOutput(mask, output, pat, sshArgs...)
}

func (c *CmdSSHActions) CopyFromHost(hostName, source, destination string) (string, error) {
	return c.cli.Execute(false, "scp", "-i", c.sshKey, "-o", "LogLevel=error", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%v@%v:%v", c.sshUser, hostName, source), destination)
}

func (c *CmdSSHActions) CopyToHost(hostName, source, destination string) (string, error) {
	if c.sudoUser == "" {
		return c.cli.Execute(false, "scp", "-i", c.sshKey, "-o", "LogLevel=error", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", source, fmt.Sprintf("%v@%v:%v", c.sshUser, hostName, destination))
	}
	// have to do something more complex in this case and _unfortunately_ copy to the /tmp dir
	tmpFile := "/tmp/transfer_file"
	out, err := c.cli.Execute(false, "scp", "-i", c.sshKey, "-o", "LogLevel=error", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", source, fmt.Sprintf("%v@%v:%v", c.sshUser, hostName, tmpFile))
	if err != nil {
		return out, err
	}
	defer func() {
		out, err := c.HostExecute(false, hostName, "rm", tmpFile)
		if err != nil {
			simplelog.Warningf("failed to remove file %v on node %v: %v - %v", tmpFile, hostName, err, out)
		}
	}()
	// now we can move it to it's final destination
	return c.HostExecute(false, hostName, "cp", tmpFile, destination)
}

func (c *CmdSSHActions) HostExecute(mask bool, hostName string, args ...string) (string, error) {
	var out strings.Builder
	writer := func(line string) {
		out.WriteString(line)
	}
	err := c.HostExecuteAndStream(mask, hostName, writer, "", args...)
	return out.String(), err
}

func (c *CmdSSHActions) addSSHUser(arguments []string) []string {
	if c.sudoUser == "" {
		return arguments
	}
	arguments = append(arguments, "sudo")
	arguments = append(arguments, "-u")
	arguments = append(arguments, c.sudoUser)
	return arguments
}

func CleanOut(out string) string {
	//we expect there it be a warning with ssh that we will clean here
	// Create a scanner to split the output into lines
	scanner := bufio.NewScanner(strings.NewReader(out))

	var lines []string
	var counter int
	// Iterate over each line but skip the first one due to the Warning which is always present when using ssh
	for scanner.Scan() {
		if counter > 0 {
			lines = append(lines, scanner.Text())
		}
		counter++
	}
	cleanedOut := strings.Join(lines, "\n")
	return cleanedOut
}

func (c *CmdSSHActions) GetExecutors() (hosts []string, err error) {
	return c.findHosts(c.executorStr)
}

func (c *CmdSSHActions) GetCoordinators() (hosts []string, err error) {
	return c.findHosts(c.coordinatorStr)
}

func (c *CmdSSHActions) findHosts(searchTerm string) (hosts []string, err error) {
	rawHosts := strings.Split(searchTerm, ",")
	for _, host := range rawHosts {
		if host == "" {
			continue
		}
		hosts = append(hosts, strings.TrimSpace(host))
	}
	return hosts, nil
}

func (c *CmdSSHActions) HelpText() string {
	return "no hosts found did you specify a comma separated list for the ssh-hosts? Something like: ddc --coordinator 192.168.1.10,192.168.1.11 --excecutors 192.168.1.14,192.168.1.15"
}
