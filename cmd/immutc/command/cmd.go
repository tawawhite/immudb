package immutc

import (
	"github.com/codenotary/immudb/cmd/docs/man"
	c "github.com/codenotary/immudb/cmd/helper"
	"github.com/codenotary/immudb/cmd/version"
	"github.com/codenotary/immudb/pkg/client"
	"github.com/codenotary/immudb/pkg/logger"
	"github.com/codenotary/immudb/pkg/tc"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	daem "github.com/takama/daemon"
	"os"
	"path/filepath"
)

var o = c.Options{}

func init() {
	cobra.OnInitialize(func() { o.InitConfig("immugw") })
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "immutc",
		Short: "immu trust checker: continuously launch consistency checks of random data",
		Long: `immu trust checker: continuously launch consistency checks of random data.

Environment variables:
  IMMUTC_ADDRESS=127.0.0.1
  IMMUTC_PORT=3323
  IMMUTC_IMMUDB-ADDRESS=127.0.0.1
  IMMUTC_IMMUDB-PORT=3322
  IMMUTC_PIDFILE=
  IMMUTC_LOGFILE=
  IMMUTC_DETACHED=false
  IMMUTC_MTLS=false
  IMMUTC_SERVERNAME=localhost
  IMMUTC_PKEY=./tools/mtls/4_client/private/localhost.key.pem
  IMMUTC_CERTIFICATE=./tools/mtls/4_client/certs/localhost.cert.pem
  IMMUTC_CLIENTCAS=./tools/mtls/2_intermediate/certs/ca-chain.cert.pem`,
		RunE: Immutc,
	}

	setupFlags(cmd, tc.DefaultOptions(), tc.DefaultMTLsOptions())

	if err := bindFlags(cmd); err != nil {
		c.QuitToStdErr(err)
	}
	setupDefaults(tc.DefaultOptions(), tc.DefaultMTLsOptions())
	cmd.AddCommand(man.Generate(cmd, "immutc", "./cmd/docs/man/immutc"))

	cmd.AddCommand(version.VersionCmd())

	return cmd
}

func Immutc(cmd *cobra.Command, args []string) (err error) {
	var options tc.Options
	if options, err = parseOptions(cmd); err != nil {
		return err
	}
	immutcServer := tc.
		DefaultServer().
		WithOptions(options)
	if options.Logfile != "" {
		if flogger, file, err := logger.NewFileLogger("immutc ", options.Logfile); err == nil {
			defer func() {
				if err = file.Close(); err != nil {
					c.QuitToStdErr(err)
				}
			}()
			immutcServer.WithLogger(flogger)
		} else {
			return err
		}
	}
	if options.Detached {
		c.Detached()
	}

	var d daem.Daemon
	if d, err = daem.New("immutc", "immutc", "immutc"); err != nil {
		c.QuitToStdErr(err)
	}

	service := tc.Service{
		*immutcServer,
	}

	d.Run(service)

	return
}
func parseOptions(cmd *cobra.Command) (options tc.Options, err error) {
	port := viper.GetInt("port")
	address := viper.GetString("address")
	immudbport := viper.GetInt("immudb-port")
	immudbAddress := viper.GetString("immudb-address")
	// config file came only from arguments or default folder
	if o.CfgFn, err = cmd.Flags().GetString("config"); err != nil {
		return tc.Options{}, err
	}
	pidfile := viper.GetString("pidfile")
	logfile := viper.GetString("logfile")
	mtls := viper.GetBool("mtls")
	detached := viper.GetBool("detached")
	servername := viper.GetString("servername")
	certificate := viper.GetString("certificate")
	pkey := viper.GetString("pkey")
	clientcas := viper.GetString("clientcas")

	options = tc.DefaultOptions().
		WithPort(port).
		WithAddress(address).
		WithImmudbAddress(immudbAddress).
		WithImmudbPort(immudbport).
		WithPidfile(pidfile).
		WithLogfile(logfile).
		WithMTLs(mtls).
		WithDetached(detached)
	if mtls {
		// todo https://golang.org/src/crypto/x509/root_linux.go
		options.MTLsOptions = client.DefaultMTLsOptions().
			WithServername(servername).
			WithCertificate(certificate).
			WithPkey(pkey).
			WithClientCAs(clientcas)
	}
	return options, nil
}

func setupFlags(cmd *cobra.Command, options tc.Options, mtlsOptions tc.MTLsOptions) {
	cmd.Flags().IntP("port", "p", options.Port, "immutc port number")
	cmd.Flags().StringP("address", "a", options.Address, "immutc host address")
	cmd.Flags().IntP("immudb-port", "j", options.ImmudbPort, "immudb port number")
	cmd.Flags().StringP("immudb-address", "k", options.ImmudbAddress, "immudb host address")
	cmd.Flags().StringVar(&o.CfgFn, "config", "", "config file (default path are configs or $HOME. Default filename is immutc.toml)")
	cmd.Flags().String("pidfile", options.Pidfile, "pid path with filename. E.g. /var/run/immutc.pid")
	cmd.Flags().String("logfile", options.Logfile, "log path with filename. E.g. /tmp/immutc/immutc.log")
	cmd.Flags().BoolP("mtls", "m", options.MTLs, "enable mutual tls")
	cmd.Flags().BoolP(c.DetachedFlag, c.DetachedShortFlag, options.Detached, "run immudb in background")
	cmd.Flags().String("servername", mtlsOptions.Servername, "used to verify the hostname on the returned certificates")
	cmd.Flags().String("certificate", mtlsOptions.Certificate, "server certificate file path")
	cmd.Flags().String("pkey", mtlsOptions.Pkey, "server private key path")
	cmd.Flags().String("clientcas", mtlsOptions.ClientCAs, "clients certificates list. Aka certificate authority")
}

func bindFlags(cmd *cobra.Command) error {
	if err := viper.BindPFlag("port", cmd.Flags().Lookup("port")); err != nil {
		return err
	}
	if err := viper.BindPFlag("address", cmd.Flags().Lookup("address")); err != nil {
		return err
	}
	if err := viper.BindPFlag("immudb-port", cmd.Flags().Lookup("immudb-port")); err != nil {
		return err
	}
	if err := viper.BindPFlag("immudb-address", cmd.Flags().Lookup("immudb-address")); err != nil {
		return err
	}
	if err := viper.BindPFlag("pidfile", cmd.Flags().Lookup("pidfile")); err != nil {
		return err
	}
	if err := viper.BindPFlag("logfile", cmd.Flags().Lookup("logfile")); err != nil {
		return err
	}
	if err := viper.BindPFlag("mtls", cmd.Flags().Lookup("mtls")); err != nil {
		return err
	}
	if err := viper.BindPFlag("detached", cmd.Flags().Lookup("detached")); err != nil {
		return err
	}
	if err := viper.BindPFlag("servername", cmd.Flags().Lookup("servername")); err != nil {
		return err
	}
	if err := viper.BindPFlag("certificate", cmd.Flags().Lookup("certificate")); err != nil {
		return err
	}
	if err := viper.BindPFlag("pkey", cmd.Flags().Lookup("pkey")); err != nil {
		return err
	}
	if err := viper.BindPFlag("clientcas", cmd.Flags().Lookup("clientcas")); err != nil {
		return err
	}
	return nil
}

func setupDefaults(options tc.Options, mtlsOptions tc.MTLsOptions) {
	viper.SetDefault("port", options.Port)
	viper.SetDefault("address", options.Address)
	viper.SetDefault("immudb-port", options.ImmudbPort)
	viper.SetDefault("immudb-address", options.ImmudbAddress)
	viper.SetDefault("pidfile", options.Pidfile)
	viper.SetDefault("logfile", options.Logfile)
	viper.SetDefault("mtls", options.MTLs)
	viper.SetDefault("detached", options.Detached)
	viper.SetDefault("certificate", mtlsOptions.Certificate)
	viper.SetDefault("pkey", mtlsOptions.Pkey)
	viper.SetDefault("clientcas", mtlsOptions.ClientCAs)
}

func InstallManPages() error {
	header := &doc.GenManHeader{
		Title:   "immutc",
		Section: "1",
		Source:  "Generated by immutc command",
	}
	dir := c.LinuxManPath

	_ = os.Mkdir(dir, os.ModePerm)
	err := doc.GenManTree(NewCmd(), header, dir)
	if err != nil {
		return err
	}
	return nil
}

func UnistallManPages() error {
	return os.Remove(filepath.Join(c.LinuxManPath, "immutc.1"))
}
