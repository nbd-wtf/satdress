package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cockroachdb/pebble"
	"github.com/gorilla/mux"
)

// The BOLT 11 example invoice (payment hash 0001020304...0102).
const specInvoice = "lnbc1pvjluezsp5zyg3zyg3zyg3zyg3zyg3zyg3zyg3zyg3zyg3zyg3zyg3zyg3zygspp5qqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqypqdpl2pkx2ctnv5sxxmmwwd5kgetjypeh2ursdae8g6twvus8g6rfwvs8qun0dfjkxaq9qrsgq357wnc5r2ueh7ck6q93dj32dlqnls087fxdwk8qakdyafkq3yap9us6v52vjjsrvywa6rt52cm9r9zqt8r2t7mlcwspyetp5h2tztugp9lfyql"

func key(b byte) *btcec.PrivateKey {
	raw := make([]byte, 32)
	raw[31] = b
	sk, _ := btcec.PrivKeyFromBytes(raw)
	return sk
}

func zapRequest(t *testing.T, sender *btcec.PrivateKey, tags [][]string) (string, *NostrEvent) {
	ev := &NostrEvent{CreatedAt: 1700000000, Kind: zapRequestKind, Tags: tags, Content: "thanks ⚡"}
	if err := ev.Sign(sender); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(ev)
	return string(raw), ev
}

func validTags(recipient string) [][]string {
	return [][]string{{"p", recipient}, {"amount", "21000"}, {"relays", "wss://relay.example", "wss://nos.example", "http://errado"}}
}

func TestPaymentHashFromSpecInvoice(t *testing.T) {
	h, err := paymentHash(specInvoice)
	if err != nil || h != "0001020304050607080900010203040506070809000102030405060708090102" {
		t.Fatalf("hash %q err %v", h, err)
	}
	if _, err := paymentHash("lnbc1qqqq"); err == nil {
		t.Fatal("short invoice accepted")
	}
}

func TestSerializeAndSignature(t *testing.T) {
	ev := &NostrEvent{PubKey: "ab", CreatedAt: 1, Kind: 1, Content: "a\"b<c>&\u2028"}
	if got := string(ev.serialize()); got != "[0,\"ab\",1,1,[],\"a\\\"b<c>&\u2028\"]" {
		t.Fatalf("serialization %s", got)
	}
	_, signed := zapRequest(t, key(7), validTags(xOnly(key(9))))
	if !signed.CheckSignature() {
		t.Fatal("valid signature refused")
	}
	signed.Content = "outro"
	if signed.CheckSignature() {
		t.Fatal("tampered event accepted")
	}
}

func TestZapRequestValidation(t *testing.T) {
	recipient := xOnly(key(9))
	raw, _ := zapRequest(t, key(7), validTags(recipient))
	if _, err := parseZapRequest(raw, 21000); err != nil {
		t.Fatalf("valid request refused: %v", err)
	}
	if _, err := parseZapRequest(raw, 1000); err == nil {
		t.Fatal("different amount accepted")
	}
	two, _ := zapRequest(t, key(7), append(validTags(recipient), []string{"p", recipient}))
	if _, err := parseZapRequest(two, 21000); err == nil {
		t.Fatal("two p tags accepted")
	}
	norelay, _ := zapRequest(t, key(7), [][]string{{"p", recipient}})
	if _, err := parseZapRequest(norelay, 21000); err == nil {
		t.Fatal("no relays accepted")
	}
	tampered := strings.Replace(raw, "thanks", "THANKS", 1)
	if _, err := parseZapRequest(tampered, 21000); err == nil {
		t.Fatal("tampered request accepted")
	}
	notzap := strings.Replace(raw, `"kind":9734`, `"kind":1`, 1)
	if _, err := parseZapRequest(notzap, 21000); err == nil {
		t.Fatal("wrong kind accepted")
	}
}

func TestReceipt(t *testing.T) {
	nostrKey, nostrPubkey = key(5), xOnly(key(5))
	defer func() { nostrKey, nostrPubkey = nil, "" }()
	recipient := xOnly(key(9))
	tags := append(validTags(recipient), []string{"e", strings.Repeat("e", 64)})
	raw, req := zapRequest(t, key(7), tags)
	receipt, err := zapReceipt(req, raw, specInvoice, strings.Repeat("ab", 32), 1700000100)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Kind != 9735 || receipt.PubKey != nostrPubkey || !receipt.CheckSignature() || receipt.CreatedAt != 1700000100 {
		t.Fatalf("invalid receipt: %+v", receipt)
	}
	want := map[string]string{"p": recipient, "e": strings.Repeat("e", 64), "P": req.PubKey, "bolt11": specInvoice,
		"description": raw, "preimage": strings.Repeat("ab", 32)}
	for name, value := range want {
		if tag := receipt.firstTag(name); len(tag) < 2 || tag[1] != value {
			t.Fatalf("tag %s: %v", name, tag)
		}
	}
}

func TestRelaysFilterAndExtra(t *testing.T) {
	s.ZapRelays = "wss://extra.example, wss://relay.example"
	defer func() { s.ZapRelays = "" }()
	_, req := zapRequest(t, key(7), validTags(xOnly(key(9))))
	got := strings.Join(zapRelays(req), " ")
	if got != "wss://relay.example wss://nos.example wss://extra.example" {
		t.Fatalf("relays %s", got)
	}
}

func TestCheckPaidLND(t *testing.T) {
	mac := []byte{1, 2, 3}
	pre := make([]byte, 32)
	pre[0] = 9
	settled := false
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/invoice/"+strings.Repeat("aa", 32) || r.Header.Get("Grpc-Metadata-macaroon") != "010203" {
			w.WriteHeader(401)
			return
		}
		state := "OPEN"
		if settled {
			state = "SETTLED"
		}
		json.NewEncoder(w).Encode(map[string]string{"state": state, "r_preimage": base64.StdEncoding.EncodeToString(pre), "settle_date": "1700000200"})
	}))
	defer srv.Close()
	params := &Params{Kind: "lnd", Host: srv.URL, Key: base64.StdEncoding.EncodeToString(mac)}
	if paid, _, _, err := checkPaid(params, strings.Repeat("aa", 32)); err != nil || paid {
		t.Fatalf("open invoice reported as paid: %v %v", paid, err)
	}
	settled = true
	paid, preimage, at, err := checkPaid(params, strings.Repeat("aa", 32))
	if err != nil || !paid || preimage != hex.EncodeToString(pre) || at != 1700000200 {
		t.Fatalf("paid: %v %s %d %v", paid, preimage, at, err)
	}
}

func TestCheckPaidLNbits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "chave" {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"paid": true, "preimage": strings.Repeat("0", 64)})
	}))
	defer srv.Close()
	paid, preimage, _, err := checkPaid(&Params{Kind: "lnbits", Host: srv.URL, Key: "chave"}, strings.Repeat("aa", 32))
	if err != nil || !paid || preimage != "" { // an all-zero preimage (LNbits internal invoice) is left out of the receipt
		t.Fatalf("lnbits: %v %q %v", paid, preimage, err)
	}
}

func TestWatchPublishesOnlyAfterPayment(t *testing.T) {
	nostrKey, nostrPubkey = key(5), xOnly(key(5))
	calls := 0
	var mu sync.Mutex
	published := make(chan string, 4)
	zapSleep = func(time.Duration) {}
	zapCheckPaid = func(*Params, string) (bool, string, int64, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return calls >= 3, "", 0, nil
	}
	zapPublish = func(relay string, ev *NostrEvent, _ time.Duration) error {
		if ev.Kind != 9735 || !ev.CheckSignature() {
			t.Error("invalid receipt published")
		}
		published <- relay
		return nil
	}
	defer func() {
		nostrKey, nostrPubkey = nil, ""
		zapSleep, zapCheckPaid, zapPublish = time.Sleep, checkPaid, publishEvent
	}()
	raw, req := zapRequest(t, key(7), validTags(xOnly(key(9))))
	if !watchZap(&Params{Kind: "lnd", Name: "ana"}, req, raw, specInvoice) {
		t.Fatal("zap not watched")
	}
	got := []string{<-published, <-published}
	if strings.Join(got, " ") != "wss://relay.example wss://nos.example" || calls != 3 {
		t.Fatalf("published to %v after %d lookups", got, calls)
	}
}

func TestLNURLAnnouncesZapsOnlyWhenPossible(t *testing.T) {
	dir, _ := os.MkdirTemp("", "satdress")
	defer os.RemoveAll(dir)
	var err error
	db, err = pebble.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s.Domain, s.Secret = "pay.example", "x"
	for name, kind := range map[string]string{"ana": "lnd", "bia": "sparko"} {
		data, _ := json.Marshal(Params{Kind: kind, Host: "https://node.example"})
		db.Set([]byte(getID(name, "pay.example")), data, pebble.Sync)
	}
	call := func(name, query string) map[string]interface{} {
		r := httptest.NewRequest("GET", "https://pay.example/.well-known/lnurlp/"+name+query, nil)
		r = mux.SetURLVars(r, map[string]string{"user": name})
		w := httptest.NewRecorder()
		handleLNURL(w, r)
		var out map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	if _, ok := call("ana", "")["allowsNostr"]; ok {
		t.Fatal("announced zaps without a key")
	}
	nostrKey, nostrPubkey = key(5), xOnly(key(5))
	defer func() { nostrKey, nostrPubkey = nil, "" }()
	if out := call("ana", ""); out["allowsNostr"] != true || out["nostrPubkey"] != nostrPubkey || out["tag"] != "payRequest" {
		t.Fatalf("lnd without zaps: %v", out)
	}
	if _, ok := call("bia", "")["allowsNostr"]; ok {
		t.Fatal("announced zaps for a backend we cannot query")
	}
	raw, _ := zapRequest(t, key(7), validTags(xOnly(key(9))))
	bad := strings.Replace(raw, "thanks", "THANKS", 1)
	if out := call("ana", "?amount=21000&nostr="+url.QueryEscape(bad)); out["status"] != "ERROR" {
		t.Fatalf("tampered request produced an invoice: %v", out)
	}
}
