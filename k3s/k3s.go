package k3s

import (
	"context"
	"errors"
	"fmt"
	"github.com/ertia-io/config/pkg/config"
	ertia "github.com/ertia-io/config/pkg/entities"
	"github.com/fabled-se/goph"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrorSSHNotReady = errors.New("SSH.NotReady")
)

func getServerInstallCmd(id string) string {
	return fmt.Sprintf("/tmp/%s", id)
}


func getAgentInstallCmd(nodeToken string, masterIp string, id string) string {
	return fmt.Sprintf("K3S_URL=https://%s:6443 K3S_TOKEN=%s /tmp/%s",  masterIp,strings.ReplaceAll(nodeToken,"\n",""), id)
}

func chmodInstaller(id string) string {
	return fmt.Sprintf("chmod +x /tmp/%s", id)
}

func getNodeTokenCmd() string {
	return fmt.Sprintf( "cat /var/lib/rancher/k3s/server/node-token")
}

func getK3SYamlCmd() string {
	return fmt.Sprintf( "cat /etc/rancher/k3s/k3s.yaml")
}

func UploadK3SInstaller(c *goph.Client, id string) (error) {
	ftp, err := c.NewSftp()
	if err != nil {
		return nil
	}
	defer ftp.Close()

	remote, err := ftp.Create("/tmp/"+id)
	if err != nil {
		return err
	}
	defer remote.Close()

	_, err = remote.Write([]byte(installer))

	return err
}

func InstallK3SServer(ctx context.Context, ip net.IP, user string, password string) (string, error) {

	fmt.Println("Installing K3S Server")

	sshClient,err :=tryEstablishSSHConnection(ctx,ip.String(), user, password)
	if(err!=nil){
		fmt.Println("Server not ready for SSH, retry")
		log.Ctx(ctx).Err(err).Send()
		return "", err
	}

	defer sshClient.Close()

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second * 60)
	defer cancel()

	id := ksuid.New().String()

	err = UploadK3SInstaller(sshClient, id)
	if(err!=nil){
		fmt.Println("Error:", err.Error())
		return "",err
	}

	out, err := sshClient.RunContextEscalated(timeoutCtx, chmodInstaller(id))
	if(err!=nil){
		fmt.Println("Error:", string(out))
		log.Ctx(ctx).Err(err).Msg(string(out))
		return "",err
	}


	out, err = sshClient.RunContextEscalated(timeoutCtx, getServerInstallCmd(id))
	if(err!=nil){
		fmt.Println("Err: "+err.Error())
		fmt.Println(string(out))
		log.Ctx(ctx).Err(err).Msg(string(out))
		return string(out),errors.New(fmt.Sprintf("Error: %s", out))
	}

	nodeToken, err := sshClient.RunContextEscalated(timeoutCtx, getNodeTokenCmd())
	if(err!=nil){

		fmt.Println("OUT ",string(nodeToken))
		log.Ctx(ctx).Err(err).Msg(string(nodeToken))
		return "",err
	}


	out, err = sshClient.RunContextEscalated(timeoutCtx, getK3SYamlCmd())
	if(err!=nil){

		fmt.Println("Could not fetch K3S Yaml ",string(out))
		log.Ctx(ctx).Err(err).Msg(string(out))
		return "",errors.New(fmt.Sprintf("Error: %s", out))
	}

	out = []byte(strings.ReplaceAll(string(out),"127.0.0.1", ip.String()))

	err = ioutil.WriteFile(config.ErtiaKubePath()+"/config", out, 0600)

	if(err!=nil){
		fmt.Println("Could not write Kubeconfig Yaml ",string(out))
		log.Ctx(ctx).Err(err).Msg(string(out))
		return "",err
	}
	fmt.Println("NT: ", nodeToken)

	return string(nodeToken), nil
}

func InstallK3SAgent(ctx context.Context, node ertia.Node, masterIp string) (error) {
	fmt.Println("Installing K3S Agent")
	sshClient,err :=tryEstablishSSHConnection(ctx,node.IPV4.String(), node.InstallUser, node.InstallPassword)
	if(err!=nil){
		fmt.Println("Server not ready for SSH, retry")
		log.Ctx(ctx).Err(err).Send()
		return err
	}

	defer sshClient.Close()

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second * 60)
	defer cancel()

	id := ksuid.New().String()


	err = UploadK3SInstaller(sshClient, id)
	if(err!=nil){
		fmt.Println("Error:", err.Error())
		return err
	}

	out, err := sshClient.RunContextEscalated(timeoutCtx, chmodInstaller(id))
	if(err!=nil){
		fmt.Println("Error:", string(out))
		log.Ctx(ctx).Err(err).Msg(string(out))
		return err
	}


	out, err = sshClient.RunContextEscalated(timeoutCtx, getAgentInstallCmd(node.NodeToken, masterIp, id))
	if(err!=nil){
		fmt.Println("Error:", string(out))
		log.Ctx(ctx).Err(err).Msg(string(out))
		return err
	}
	return nil
}

func tryEstablishSSHConnection(ctx context.Context,ipStr string, user string, password string) (*goph.Client,error){

	keyFiles, err := ioutil.ReadDir(config.ErtiaKeysPath())
	if(err!=nil){
		return nil, err
	}

	var client *goph.Client
	for _, keyFile := range keyFiles {

		if(strings.Contains(keyFile.Name(),".pub")){
			continue
		}
		// Start new ssh connection with private key.
		auth, err := goph.Key(filepath.Join(config.ErtiaKeysPath(),keyFile.Name()), "")
		if err != nil {
			fmt.Println(err)
			log.Ctx(ctx).Err(err).Send()
			continue
		}


		client, err = goph.NewUnknown(user, ipStr, auth)
		if err != nil {
			fmt.Println(err)
			log.Ctx(ctx).Err(err).Send()
			continue
		}

		client.SetPass(password)

		return client, nil
	}


	return nil, ErrorSSHNotReady
}

func InitK3SServer(){

}