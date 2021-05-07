package integration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	conf "github.com/tendermint/starport/starport/chainconf"
	"github.com/tendermint/starport/starport/pkg/chaintest"
	"github.com/tendermint/starport/starport/pkg/confile"
	"github.com/tendermint/starport/starport/pkg/randstr"
)

func TestOverwriteSDKConfigsAndChainID(t *testing.T) {
	var (
		env               = chaintest.New(t)
		appname           = randstr.Runes(10)
		path              = env.Scaffold(appname)
		homePath          = env.TmpDir()
		servers           = env.RandomizeServerPorts(path, "")
		ctx, cancel       = context.WithCancel(env.Ctx())
		isBackendAliveErr error
	)

	var c conf.Config

	cf := confile.New(confile.DefaultYAMLEncodingCreator, filepath.Join(path, "config.yml"))
	require.NoError(t, cf.Load(&c))

	c.Genesis = map[string]interface{}{"chain_id": "cosmos"}
	c.Init.App = map[string]interface{}{"hello": "cosmos"}
	c.Init.Config = map[string]interface{}{"fast_sync": false}

	require.NoError(t, cf.Save(c))

	go func() {
		defer cancel()
		isBackendAliveErr = env.IsAppServed(ctx, servers.Host)
	}()

	env.Must(env.Serve("should serve",
		path,
		chaintest.ServeWithHome(homePath),
		chaintest.ServeWithExecOption(chaintest.ExecCtx(ctx))),
	)

	require.NoError(t, isBackendAliveErr, "app cannot get online in time")

	configs := []struct {
		ec          confile.EncodingCreator
		relpath     string
		key         string
		expectedVal interface{}
	}{
		{confile.DefaultJSONEncodingCreator, "config/genesis.json", "chain_id", "cosmos"},
		{confile.DefaultTOMLEncodingCreator, "config/app.toml", "hello", "cosmos"},
		{confile.DefaultTOMLEncodingCreator, "config/config.toml", "fast_sync", false},
	}

	for _, c := range configs {
		var conf map[string]interface{}
		cf := confile.New(c.ec, filepath.Join(homePath, c.relpath))
		require.NoError(t, cf.Load(&conf))
		require.Equal(t, c.expectedVal, conf[c.key])
	}
}
