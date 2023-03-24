package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/xor-shift/teleserver/ingest"
	"log"
	"math/big"
)

// THESE WILL BE SECURELY STORED AND CHANGED WITH THE FIRST COMMIT THAT REMOVES THE HARD-CODED VALUES

const (
	PrivateKeyText = "145894e3c5f680ac2caab943f89e3d6f7feddeccc363c1c7dbc521c10d5dd6f0"
	PublicKeyXText = "d76176dcfe0467306b28ff89bf951d41719700bd3054ebdd153133642fb5dd23"
	PublicKeyYText = "cf52079fc23428f234f400dffeb38a4370e2d055cc5a4b98e6cf9ab4116ae8fa"
)

var (
	publicKey  ecdsa.PublicKey
	privateKey ecdsa.PrivateKey

	app *iris.Application
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("loading dotenv failed: %s", err)
	}

	publicKey = ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     big.NewInt(0),
		Y:     big.NewInt(0),
	}

	privateKey = ecdsa.PrivateKey{
		PublicKey: publicKey,
		D:         big.NewInt(0),
	}

	privateKey.D.SetString(PrivateKeyText, 16)
	privateKey.X.SetString(PublicKeyXText, 16)
	privateKey.Y.SetString(PublicKeyYText, 16)

	app = iris.New()
}

func main() {
	in, err := ingest.NewIngester(publicKey)

	if err != nil {
		log.Fatalln(err)
	}

	in.Start(1)

	app.Get("/session_reset_challenge", func(ctx iris.Context) {
		app.Logger().Printf("session reset challenge request from %s", ctx.RemoteAddr())

		resetToken := in.GetResetChallenge()
		_, _ = ctx.Text("+CST_RESET_CHALLENGE %s", resetToken)
	})

	app.Post("/session_reset_challenge", func(ctx iris.Context) {
		app.Logger().Printf("session reset request from %s", ctx.RemoteAddr())

		body, err := ctx.GetBody()
		if err != nil {
			app.Logger().Printf("session_reset_challenge (POST) error: %s", err)
		}

		if len(body) != 128 {
			_, _ = ctx.Text("+CST_RESET_FAIL 1")
			return
		}

		r := string(body[0:64])
		s := string(body[64:128])

		app.Logger().Printf("checking signature:")
		app.Logger().Printf("r = %s", r)
		app.Logger().Printf("s = %s", s)

		if err := in.ResetChallengeResponse(string(body)); err == nil {
			app.Logger().Printf("reset challenge passed, started session %d", in.SessionID())

			_, _ = ctx.Text("+CST_RESET_SUCC " + in.GetInitialRNGVector())
		} else {
			app.Logger().Warnf("reset challenge failed with error: %s", err)

			_, _ = ctx.Text("+CST_RESET_FAIL 0")
			return
		}
	})

	app.Post("/packet/full", func(ctx iris.Context) {
		body, err := ctx.GetBody()
		if err != nil {
			app.Logger().Printf("/packet/x error (body): %s", err)
		}

		//app.Logger().Printf("got a packet with body: %s", string(body))

		packets, err := ingest.ParsePackets[ingest.FullPacket](body, publicKey)

		if err != nil {
			app.Logger().Printf("/packet/x error (ParsePacket): %s", err)
			return
		}

		if err = in.NewPackets(packets); err != nil {
			app.Logger().Printf("/packet/x error (NewPacket): %s", err)
			return
		}
	})

	if err := app.Listen(":8080"); err != nil {
		log.Fatalln(err)
	}
}
