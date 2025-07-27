package e2e_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/require"
	client "github.com/sushiag/go-webrtc-signaling-server/client"
	server "github.com/sushiag/go-webrtc-signaling-server/server/server"
	sqlitedb "github.com/sushiag/go-webrtc-signaling-server/server/server/register"
)

func TestEndToEndSignaling(t *testing.T) {
	const testdata = "friends.db"
	queries, dbConn := sqlitedb.NewDatabase(testdata)

	srv, serverURL := server.StartServer("0", queries)
	defer srv.Close()

	defer func() {
		_ = dbConn.Close()
		_ = os.Remove(testdata)
	}() // ensure server is closed after the test

	httpBase := fmt.Sprintf("http://%s", serverURL)
	wsURL := fmt.Sprintf("ws://%s/ws", serverURL)

	user1 := "spongebob"
	user2 := "patrickk"
	initialPass := "initPass4ever"
	newPass := "newPass4ever"

	require.NoError(t, client.RegisterUser(httpBase, user1, initialPass))
	require.NoError(t, client.RegisterUser(httpBase, user2, initialPass))

	require.NoError(t, client.ResetPassword(httpBase, user1, initialPass, newPass))
	require.NoError(t, client.ResetPassword(httpBase, user2, initialPass, newPass))

	apiKeyA, err := client.RegenerateAPIKey(httpBase, user1, newPass)
	require.NotEmpty(t, apiKeyA)

	require.NoError(t, os.Setenv("API_KEY", apiKeyA))

	apiKeyB, err := client.RegenerateAPIKey(httpBase, user2, newPass)
	require.NotEmpty(t, apiKeyB)

	require.NoError(t, os.Setenv("API_KEY", apiKeyB))

	clientA, err := client.NewClientWithKey(wsURL, apiKeyA)
	require.NoError(t, err)
	t.Logf("client A connected to the signaling server")

	clientB, err := client.NewClientWithKey(wsURL, apiKeyB)
	require.NoError(t, err)
	t.Logf("client B connected to the signaling server")

	createdRoomID, err := clientA.CreateRoom()
	require.NoError(t, err, "client A failed to create room")
	t.Logf("client A created room %d", createdRoomID)

	clientsInRoom, err := clientB.JoinRoom(createdRoomID)
	require.NoError(t, err, "client B failed to join room %d", createdRoomID)
	t.Logf("client B joined room %d", createdRoomID)
	require.ElementsMatch(t, []uint64{1, 2}, clientsInRoom)

	clientAMsg := "Hello from Client A!"
	clientBMsg := "Wassup from Client B!"

	t.Logf("---- Waiting for data channels to open ----")
	readyDataChannels := 0
	dataChDeadline := time.After(5 * time.Second)
	dataChOpenedA := clientA.GetDataChOpened()
	dataChOpenedB := clientB.GetDataChOpened()
	for readyDataChannels < 2 {
		select {
		case peerID := <-dataChOpenedA:
			{
				require.Equal(t, clientB.GetClientID(), peerID)
				readyDataChannels += 1
			}
		case peerID := <-dataChOpenedB:
			{
				require.Equal(t, clientA.GetClientID(), peerID)
				readyDataChannels += 1
			}
		case <-dataChDeadline:
			{
				t.Fatal("clients took longer than 5 secs to open their data channels")
			}
		}
	}
	t.Logf("---- Data channels open! ----")

	t.Logf("---- Sending Messages Start ----")
	err = clientA.SendDataToPeer(clientB.GetClientID(), []byte(clientAMsg))
	require.NoError(t, err, "client A failed to send message")

	err = clientB.SendDataToPeer(clientA.GetClientID(), []byte(clientBMsg))
	require.NoError(t, err, "client B failed to send message")
	t.Logf("---- Sending Messages End ----")

	receivedMsgs := 0
	recvMsgDeadline := time.After(5 * time.Second)
	clientAMsgOut := clientA.GetPeerDataMsgCh()
	clientBMsgOut := clientB.GetPeerDataMsgCh()
	for receivedMsgs < 2 {
		select {
		case msg := <-clientAMsgOut:
			{
				require.Equal(t, msg.From, clientB.GetClientID())
				require.Equal(t, msg.Data, []byte(clientBMsg))
				receivedMsgs += 1
			}

		case msg := <-clientBMsgOut:
			{
				require.Equal(t, msg.From, clientA.GetClientID())
				require.Equal(t, msg.Data, []byte(clientAMsg))
				receivedMsgs += 1
			}

		case <-recvMsgDeadline:
			{
				t.Fatal("clients took longer than 5 secs to exchange messages")
			}
		}
	}

	t.Logf("All clients successfully exchanged messages for 1 round.")
}
