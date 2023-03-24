package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
	"github.com/xor-shift/teleserver/ingest"
	"log"
	"math/big"
)

var (
	cookieNameForSessionID = "sessionCookie"
	sess                   = sessions.New(sessions.Config{Cookie: cookieNameForSessionID})

	publicKey  ecdsa.PublicKey
	privateKey ecdsa.PrivateKey

	app   *iris.Application
	state *State
)

const (
	PrivateKeyText = "145894e3c5f680ac2caab943f89e3d6f7feddeccc363c1c7dbc521c10d5dd6f0"
	PublicKeyXText = "d76176dcfe0467306b28ff89bf951d41719700bd3054ebdd153133642fb5dd23"
	PublicKeyYText = "cf52079fc23428f234f400dffeb38a4370e2d055cc5a4b98e6cf9ab4116ae8fa"
)

type AuthStatus = int

const (
	Unauthorized AuthStatus = iota
	RegularClient
	AdminClient
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

	state, err = NewState()
	if err != nil {
		log.Fatalf("creating state failed: %s", err)
	}
}

func PacketEndpoint[T EssentialsPacket | FullPacket](ctx iris.Context) {
	body, err := ctx.GetBody()
	if err != nil {
		app.Logger().Printf("/packet/x error (body): %s", err)
	}

	//app.Logger().Printf("got a packet with body: %s", string(body))

	packets, err := ParsePackets[T](body)

	if err != nil {
		app.Logger().Printf("/packet/x error (ParsePacket): %s", err)
		return
	}

	if err = state.NewPackets(context.TODO(), packets); err != nil {
		app.Logger().Printf("/packet/x error (NewPacket): %s", err)
		return
	}
}

func main_old() {
	/*hash := sha256.Sum256([]byte("Hello, world!"))

	r, s, err := ecdsa.Sign(rand.Reader, &privateKey, hash[:])

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(ecdsa.Verify(&publicKey, hash[:], r, s))

	stmR, _ := big.NewInt(0).SetString("aa9e56e3e8cf4d346cbe97c56848073776935733b14f648a22cf0d321445fb84", 16)
	stmS, _ := big.NewInt(0).SetString("ca9e2a496dc09c90c821fb8f204ed6a44b191a4727674fc4889156499147e532", 16)

	fmt.Println(ecdsa.Verify(&publicKey, hash[:], stmR, stmS))*/

	app.Get("/session_reset_challenge", func(ctx iris.Context) {
		app.Logger().Printf("session reset challenge request from %s", ctx.RemoteAddr())

		resetToken := state.GetResetToken()
		_, _ = ctx.Text("+CST_RESET_CHALLENGE %064s", big.NewInt(0).SetBytes(resetToken[:]).Text(16))
	})

	app.Post("/session_reset_challenge", func(ctx iris.Context) {
		app.Logger().Printf("session reset request from %s", ctx.RemoteAddr())

		body, err := ctx.GetBody()
		if err != nil {
			app.Logger().Printf("session_reset_challenge (POST) error: %s", err)
		}

		if len(body) != 128 {
			_, _ = ctx.Text("+CST_RESET_FAIL 1")
		}

		r := string(body[0:64])
		s := string(body[64:128])

		app.Logger().Printf("checking signature:")
		app.Logger().Printf("r = %s", r)
		app.Logger().Printf("s = %s", s)

		if err := state.Reset(r, s); err == nil {
			app.Logger().Printf("reset challenge passed, started session %d", state.sessionNo)

			_, _ = ctx.Text("+CST_RESET_SUCC %08x%08x%08x%08x",
				state.initialRNGVector[0],
				state.initialRNGVector[1],
				state.initialRNGVector[2],
				state.initialRNGVector[3])
		} else {
			app.Logger().Warnf("reset challenge failed with error: %s", err)

			_, _ = ctx.Text("+CST_RESET_FAIL 0")
		}
	})

	app.Post("/packet/essentials", PacketEndpoint[EssentialsPacket])

	app.Post("/packet/full", PacketEndpoint[FullPacket])

	app.Post("/session/dataCurrent", func(ctx iris.Context) {
		session := sess.Start(ctx)

		if authStatus := session.GetIntDefault("authStatus", Unauthorized); authStatus != RegularClient && authStatus != AdminClient {
			ctx.StatusCode(iris.StatusForbidden)
			return
		}

	})

	app.Post("/session", func(ctx iris.Context) {
		session := sess.Start(ctx)

		givenKey := ctx.FormValueDefault("key", "")

		if givenKey != "this is not a secure key" {
			ctx.StatusCode(iris.StatusForbidden)
			return
		}

		session.Set("authStatus", AdminClient)
	})

	if err := app.Listen(":8080"); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	state := 0 // shadow
	(func(any) {})(state)

	in, err := ingest.NewIngester(publicKey)

	in.Start(1)

	if err != nil {
		log.Fatalln(err)
	}

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

			_, _ = ctx.Text("+CST_RESET_SUCC " + in.GetCurrentRNGVector())
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
