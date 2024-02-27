package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
)

func getHostIp() string {
	addrList, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("get current host ip err: ", err)
		return ""
	}
	var ip string
	for _, address := range addrList {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ip = ipNet.IP.String()
				break
			}
		}
	}
	return ip
}

type Host struct {
	Host string `yaml:"host"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type Machines struct {
	Machine []Host `yaml:"machine"`
}

type sshCommands struct {
	keyGen string
	copyID string
}

// 读取配置获取节点和密码
func readConfig() Machines {
	file, err := os.ReadFile("hosts.yaml")
	if err != nil {
		log.Fatalf("open file error: %v", err)
	}

	machine := Machines{}
	if err = yaml.Unmarshal(file, &machine); err != nil {
		log.Fatalf("yaml to struct error: %v", err)
	}

	return machine
}

// 判断id_rsa文件是否存在
func fileExits(user string) bool {
	_, err := os.Stat(fmt.Sprintf("/home/%s/.ssh/id_rsa.pub", user))
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// sshpass + ssh 命令组成
func genCommands(m Machines) []sshCommands {
	commands := make([]sshCommands, 0, len(m.Machine))
	localIP := getHostIp()
	for _, v := range m.Machine {
		if localIP == v.Host {
			// 判断id_rsa.pub 文件是否存在
			keyGen := ""
			if !fileExits(v.User) {
				keyGen = " -t rsa -N '' -f id_rsa -q"
			}
			commands = append(commands, sshCommands{
				keyGen: keyGen,
				copyID: fmt.Sprintf(" -p %s ssh-copy-id -i ~/.ssh/id_rsa.pub %s@%s", v.Pass, v.User, v.Host),
			})
			continue
		}
		commands = append(commands, sshCommands{
			copyID: fmt.Sprintf(" -p %s ssh-copy-id -i ~/.ssh/id_rsa.pub %s@%s", v.Pass, v.User, v.Host),
		})
	}
	for k, v := range commands {
		if v.keyGen != "" && k != 0 {
			tmp := v.copyID
			commands[0].keyGen = v.keyGen
			commands[k].copyID = commands[0].copyID
			commands[k].keyGen = ""
			commands[0].copyID = tmp
		}
	}
	return commands
}

// ssh-key生成和分发
func genSSHKey(cmds []sshCommands) {
	for _, v := range cmds {
		fmt.Println(v.keyGen)
		fmt.Println(v.copyID)
	}
	for _, v := range cmds {
		if v.keyGen != "" {
			cmd := exec.Command("ssh-keygen", v.keyGen)
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Fatalf("command: %s out: %s error: %v", v.keyGen, out, err)
			}
		}

		if v.copyID != "" {
			cmd := exec.Command("sh", "-c", fmt.Sprintf("sshpass %s", v.copyID))
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Fatalf("command: %s out: %s error: %v", v.copyID, out, err)
			}
		}
	}
}

func main() {
	m := readConfig()
	commands := genCommands(m)
	genSSHKey(commands)
}
