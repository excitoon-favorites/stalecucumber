package stalecucumber

import (
	"io"
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func testString(t *testing.T, input string, expect string) {
	reader := strings.NewReader(input)

	result, err := String(Unpickle(reader))

	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if result != expect {
		t.Fatalf("Got %q(%T) expected %q(%T)", result, result, expect, expect)
	}

}

func TestJorilxUnicodeString(t *testing.T) {
	input := []byte{0x56, 0xE0, 0x0A, // Và
		0x70, 0x30, 0x0A, // p0
		0x2E} // .
	reader := bytes.NewReader(input)
	unpickled, err := String(Unpickle(reader))
	const EXPECT = `à`
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(unpickled) != 1 {
		t.Errorf("wrong length string unpacked %d,%q", utf8.RuneCountInString(unpickled), unpickled)
	}

	if unpickled != EXPECT {
		t.Errorf("Expected %q got %q", EXPECT, unpickled)
	}

	inputProtocol2 := strings.NewReader("\x80\x02X\x02\x00\x00\x00\xc3\xa0q\x00.")

	unpickled, err = String(Unpickle(inputProtocol2))
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(unpickled) != 1 {
		t.Errorf("wrong length string unpacked %d,%q", utf8.RuneCountInString(unpickled), unpickled)
	}

	if unpickled != EXPECT {
		t.Errorf("Expected %q got %q", EXPECT, unpickled)
	}

	const EXPECT_SNOWMAN = "à ☃"

	const inputWithSnowman = "\x56\xe0\x20\x5c\x75\x32\x36\x30\x33\x0a\x70\x30\x0a\x2e"

	unpickled, err = String(Unpickle(strings.NewReader(inputWithSnowman)))

	if err != nil {
		t.Fatal(err)
	}

	if unpickled != EXPECT_SNOWMAN {
		t.Errorf("Expected %q got %q", EXPECT_SNOWMAN, unpickled)
	}

}

func TestProtocol0Integer(t *testing.T) {

	reader := strings.NewReader("I42\n.")
	result, err := Int(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	const EXPECT = 42
	if result != EXPECT {
		t.Fatalf("Got value %d expected %d", result, EXPECT)
	}
}

func TestProtocol0Bool(t *testing.T) {
	var result bool

	reader := strings.NewReader("I00\n.")
	result, err := Bool(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if result != false {
		t.Fatalf("Got value %v expected %v", result, false)
	}

	reader = strings.NewReader("I01\n.")
	result, err = Bool(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if result != true {
		t.Fatalf("Got value %v expected %v", result, true)
	}

}

func TestProtocol0String(t *testing.T) {
	testString(t, "S''\np0\n.", "")
	testString(t, "S'foobar'\np0\n.", "foobar")
	testString(t, "S\"with single '\"\np0\n.", "with single '")
	testString(t, "S'String with embedded\\nnewline.'\np0\n.", "String with embedded\nnewline.")
	testString(t,
		"\x53\x27\x53\x74\x72\x69\x6e\x67\x20\x77\x69\x74\x68\x20\x65\x6d\x62\x65\x64\x64\x65\x64\x5c\x6e\x6e\x65\x77\x6c\x69\x6e\x65\x20\x61\x6e\x64\x20\x65\x6d\x62\x65\x64\x64\x65\x64\x20\x71\x75\x6f\x74\x65\x20\x5c\x27\x20\x61\x6e\x64\x20\x65\x6d\x62\x65\x64\x64\x65\x64\x20\x64\x6f\x75\x62\x6c\x65\x71\x75\x6f\x74\x65\x20\x22\x2e\x27\x0a\x70\x30\x0a\x2e",
		"String with embedded\nnewline and embedded quote ' and embedded doublequote \".")
}

func testBigIntFromString(t *testing.T, input string, expectStr string) {
	var expect big.Int

	_, err := fmt.Sscanf(expectStr, "%d", &expect)
	if err != nil {
		t.Fatalf("got error parsing %q:%v", expectStr, err)
	}

	testBigInt(t, input, &expect)

}

func testBigInt(t *testing.T, input string, expect *big.Int) {

	reader := strings.NewReader(input)

	result, err := Big(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if result.Cmp(expect) != 0 {
		t.Fatalf("Got value %s expected %s", result, expect)
	}

}

func TestProtocol0Long(t *testing.T) {
	testBigInt(t, "L5L\n.", big.NewInt(5))
	testBigIntFromString(t, "L18446744073709551615L\n.", "18446744073709551615")
	testBigIntFromString(t, "L-18446744073709551615L\n.", "-18446744073709551615")
}

func TestProtocol0Float(t *testing.T) {

	reader := strings.NewReader("F3.14\n.")
	const EXPECT = 3.14

	result, err := Float(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if EXPECT != result {
		t.Fatalf("Got value %q expected %q", result, EXPECT)
	}
}

func testDict(t *testing.T, input string, expect map[interface{}]interface{}) {
	reader := strings.NewReader(input)

	result, err := Dict(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if len(result) != len(expect) {
		t.Errorf("result has wrong length %d", len(result))
	}

	for k, v := range result {
		var expectedV interface{}

		expectedV, ok := expect[k]
		if !ok {
			t.Errorf("Result has key %v(%T) which is not in expectation", k, k)
			continue
		}

		if reflect.TypeOf(v) != reflect.TypeOf(expectedV) {
			t.Errorf("At key %v result has type %T where expectation has type %T", k, v, expectedV)
			continue
		}

		if !reflect.DeepEqual(expectedV, v) {
			t.Errorf("At key %v result %v != expectation %v", k, v, expectedV)
		}

	}
}

func TestProtocol0Get(t *testing.T) {
	testList(t, "(lp0\nS'hydrogen18'\np1\nag1\na.", []interface{}{"hydrogen18", "hydrogen18"})
}

func TestProtocol1Get(t *testing.T) {
	testList(t, "]q\x00(U\nhydrogen18q\x01h\x01e.", []interface{}{"hydrogen18", "hydrogen18"})
}

func TestProtocol0Dict(t *testing.T) {

	{
		input := "(dp0\nS'a'\np1\nI1\nsS'b'\np2\nI5\ns."
		expect := make(map[interface{}]interface{})
		expect["a"] = int64(1)
		expect["b"] = int64(5)
		testDict(t, input, expect)
	}

	{
		expect := make(map[interface{}]interface{})
		expect["foo"] = "bar"
		expect[int64(5)] = "kitty"
		expect["num"] = 13.37
		expect["list"] = []interface{}{int64(1), int64(2), int64(3), int64(4)}
		testDict(t, "(dp0\nS'list'\np1\n(lp2\nI1\naI2\naI3\naI4\nasS'foo'\np3\nS'bar'\np4\nsS'num'\np5\nF13.37\nsI5\nS'kitty'\np6\ns.", expect)
	}

}

func TestProtocol0Set(t *testing.T) {
	// pickle.dumps(set(['a','b']))
	reader := strings.NewReader("c__builtin__\nset\np0\n((lp1\nS'a'\np2\naS'b'\np3\natp4\nRp5\n.")
	result, err := Set(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Got unexpected item count in set, result %v != expectation %v", len(result), 2)
	}

	if !result["a"] {
		t.Errorf("Expected item 'a' in set")
	}

	if !result["b"] {
		t.Errorf("Expected item 'b' in set")
	}
}

func TestProtocol1Set(t *testing.T) {
	// pickle.dumps(set(['a','b']))
	reader := strings.NewReader("c__builtin__\nset\nq\x00(]q\x01(U\x01aq\x02U\x01bq\x03etq\x04Rq\x05.")
	result, err := Set(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Got unexpected item count in set, result %v != expectation %v", len(result), 2)
	}

	if !result["a"] {
		t.Errorf("Expected item 'a' in set")
	}

	if !result["b"] {
		t.Errorf("Expected item 'b' in set") 
	}
}

func TestProtocol2Set(t *testing.T) {
	// pickle.dumps(set(['a','b']))
	reader := strings.NewReader("\x80\x02c__builtin__\nset\nq\x00]q\x01(U\x01aq\x02U\x01bq\x03e\x85q\x04Rq\x05.")
	result, err := Set(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Got unexpected item count in set, result %v != expectation %v", len(result), 2)
	}

	if !result["a"] {
		t.Errorf("Expected item 'a' in set")
	}

	if !result["b"] {
		t.Errorf("Expected item 'b' in set")
	}
}

func TestProtocol0GarbageReduce(t *testing.T){
	reader := strings.NewReader("S'foo'\nS'bar'\nR.")
	/**
	Disassembly of the above is this garbage
	0: S    STRING     'foo'
	7: S    STRING     'bar'
 14: R    REDUCE
 15: .    STOP
 **/

 result, err := Unpickle(reader)
 if err == nil {
	 t.Error("Expected an error")
 }

 if result != nil {
	 t.Errorf("Expected result to be nil but got %v", result)
 }

 // Unpack the generic error type to get the actual one
 err = (err.(PickleMachineError)).Err

 if expectedErr, ok := err.(UnreducibleValueError); !ok {
	 t.Errorf("Expected error of type %T but got %T %v", expectedErr, err, err)
 }
}

func TestProtocol0Bytearray(t *testing.T){
	reader := strings.NewReader("c__builtin__\nbytearray\np0\n(Vabc123\np1\nS'latin-1'\np2\ntp3\nRp4\n.")

	result, err := Unpickle(reader)
	if err != nil{
		t.Errorf("Expected no error but got %v", err)
	}

	if result == nil {
		t.Error("Expected result value but got nil")
	}

	buffer, ok := result.(*strings.Reader)	
	if !ok{
		t.Errorf("Expected byte buffer but got %T", result)
	}

	const expected = `abc123`
	actual := bytes.NewBuffer(nil)
	_, err = io.Copy(actual, buffer)
	if err != nil {
		t.Fatal(err)
	}
	
	if actual.String() != expected {
		t.Errorf("Expected %q but got %q", expected, actual.String())
	}

}

func TestProtocol1Dict(t *testing.T) {
	testDict(t, "}q\x00.", make(map[interface{}]interface{}))
	{
		expect := make(map[interface{}]interface{})
		expect["foo"] = "bar"
		expect["meow"] = "bar"
		expect[int64(5)] = "kitty"
		expect["num"] = 13.37
		expect["list"] = []interface{}{int64(1), int64(2), int64(3), int64(4)}
		input := "}q\x00(U\x04meowq\x01U\x03barq\x02U\x04listq\x03]q\x04(K\x01K\x02K\x03K\x04eU\x03fooq\x05h\x02U\x03numq\x06G@*\xbdp\xa3\xd7\n=K\x05U\x05kittyq\x07u."
		testDict(t, input, expect)
	}
}

func testListsEqual(t *testing.T, result []interface{}, expect []interface{}) {
	if len(result) != len(expect) {
		t.Errorf("Result has wrong length %d", len(result))
	}
	for i, v := range result {

		vexpect := expect[i]

		if !reflect.DeepEqual(v, vexpect) {
			t.Errorf("result[%v](%T) != expect[%v](%T)", i, v, i, vexpect)
			t.Errorf("result[%d]=%v", i, v)
			t.Errorf("expect[%d]=%v", i, vexpect)
		}

	}
}

func testList(t *testing.T, input string, expect []interface{}) {

	reader := strings.NewReader(input)

	result, err := ListOrTuple(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	testListsEqual(t, result, expect)

}

func TestProtocol0List(t *testing.T) {
	testList(t, "(lp0\nI1\naI2\naI3\na.", []interface{}{int64(1), int64(2), int64(3)})
}

func TestProtocol1List(t *testing.T) {
	testList(t, "]q\x00.", []interface{}{})
	testList(t, "]q\x00(M9\x05M9\x05M9\x05e.", []interface{}{int64(1337), int64(1337), int64(1337)})
	testList(t, "]q\x00(M9\x05I3735928559\nM\xb1\"e.", []interface{}{int64(1337), int64(0xdeadbeef), int64(8881)})
}

func TestProtocol1Tuple(t *testing.T) {
	testList(t, ").", []interface{}{})
	testList(t, "(K*K\x18K*K\x1cKRK\x1ctq\x00.", []interface{}{int64(42), int64(24), int64(42), int64(28), int64(82), int64(28)})
}

func testInt(t *testing.T, input string, expect int64) {
	reader := strings.NewReader(input)

	result, err := Int(Unpickle(reader))
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if result != expect {
		t.Fatalf("Got %d(%T) expected %d(%T)", result, result, expect, expect)
	}

}

func TestProtocol1Binint(t *testing.T) {
	testInt(t, "J\xff\xff\xff\x00.", 0xffffff)
	testInt(t, "K*.", 42)
	testInt(t, "M\xff\xab.", 0xabff)
}

func TestProtocol1String(t *testing.T) {
	testString(t, "U\x00q\x00.", "")
	testString(t,
		"T\x04\x01\x00\x00abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZq\x00.",
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	testString(t, "U\x13queen of the castleq\x00.", "queen of the castle")
}

func TestProtocol1Float(t *testing.T) {

	reader := strings.NewReader("G?\xc1\x1d\x14\xe3\xbc\xd3[.")

	result, err := Float(Unpickle(reader))

	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	var expect float64
	expect = 0.1337
	if result != expect {
		t.Fatalf("Got %f expected %f", result, expect)
	}
}

func TestProtocol1PopMark(t *testing.T) {
	/**
		This exapmle is ultra-contrived. I could not get anything to
		produce usage of POP_MARK using protocol 1. There are some
		comments in Lib/pickle.py about a recursive tuple generating
		this but I have no idea how that is even possible.

		The disassembly of this is
	    0: K    BININT1    1
	    2: (    MARK
	    3: K        BININT1    2
	    5: K        BININT1    3
	    7: 1        POP_MARK   (MARK at 2)
	    8: .    STOP

		There is just a mark placed on the stack with some numbers
		afterwards solely to test the correct behavior
		of the POP_MARK instruction.

		**/
	reader := strings.NewReader("K\x01(K\x02K\x031.")
	result, err := Int(Unpickle(reader))

	const EXPECT = 1
	if err != nil {
		t.Fatalf("Got error %v", err)
	}

	if EXPECT != result {
		t.Fatalf("Got %d expected %d", result, EXPECT)
	}
}

func TestProtocol1Unicode(t *testing.T) {
	testString(t, "X\x00\x00\x00\x00q\x00.", "")

	expect := "This is a slash \\. This is a newline \n. This is a character that is two embedded newlines: \u0a0a. This is a snowman: \u2603."

	if len([]rune(expect)) != 115 {
		t.Errorf("Expect shouldn't be :%v", expect)
		t.Fatalf("you messed up the escape sequence on the expecation, again. Length is %d", len(expect))
	}

	testString(t, "\x58\x77\x00\x00\x00\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x73\x6c\x61\x73\x68\x20\x5c\x2e\x20\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x6e\x65\x77\x6c\x69\x6e\x65\x20\x0a\x2e\x20\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x63\x68\x61\x72\x61\x63\x74\x65\x72\x20\x74\x68\x61\x74\x20\x69\x73\x20\x74\x77\x6f\x20\x65\x6d\x62\x65\x64\x64\x65\x64\x20\x6e\x65\x77\x6c\x69\x6e\x65\x73\x3a\x20\xe0\xa8\x8a\x2e\x20\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x73\x6e\x6f\x77\x6d\x61\x6e\x3a\x20\xe2\x98\x83\x2e\x71\x00\x2e",
		expect)

}

func TestProtocol0Unicode(t *testing.T) {

	testString(t, "V\np0\n.", "")

	expect := "This is a slash \\. This is a newline \n. This is a character that is two embedded newlines: \u0a0a. This is a snowman: \u2603."

	if len([]rune(expect)) != 115 {
		t.Errorf("Expect shouldn't be :%v", expect)
		t.Fatalf("you messed up the escape sequence on the expecation, again. Length is %d", len(expect))
	}
	testString(t, "\x56\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x73\x6c\x61\x73\x68\x20\x5c\x75\x30\x30\x35\x63\x2e\x20\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x6e\x65\x77\x6c\x69\x6e\x65\x20\x5c\x75\x30\x30\x30\x61\x2e\x20\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x63\x68\x61\x72\x61\x63\x74\x65\x72\x20\x74\x68\x61\x74\x20\x69\x73\x20\x74\x77\x6f\x20\x65\x6d\x62\x65\x64\x64\x65\x64\x20\x6e\x65\x77\x6c\x69\x6e\x65\x73\x3a\x20\x5c\x75\x30\x61\x30\x61\x2e\x20\x54\x68\x69\x73\x20\x69\x73\x20\x61\x20\x73\x6e\x6f\x77\x6d\x61\x6e\x3a\x20\x5c\x75\x32\x36\x30\x33\x2e\x0a\x70\x30\x0a\x2e",
		expect)

}

func TestProtocol1LongBinPutBinGet(t *testing.T) {
	input := "\x5d\x71\x00\x28\x5d\x71\x01\x28\x55\x01\x30\x71\x02\x55\x01\x31\x71\x03\x55\x01\x32\x71\x04\x55\x01\x33\x71\x05\x55\x01\x34\x71\x06\x55\x01\x35\x71\x07\x55\x01\x36\x71\x08\x55\x01\x37\x71\x09\x55\x01\x38\x71\x0a\x55\x01\x39\x71\x0b\x55\x02\x31\x30\x71\x0c\x55\x02\x31\x31\x71\x0d\x55\x02\x31\x32\x71\x0e\x55\x02\x31\x33\x71\x0f\x55\x02\x31\x34\x71\x10\x55\x02\x31\x35\x71\x11\x55\x02\x31\x36\x71\x12\x55\x02\x31\x37\x71\x13\x55\x02\x31\x38\x71\x14\x55\x02\x31\x39\x71\x15\x55\x02\x32\x30\x71\x16\x55\x02\x32\x31\x71\x17\x55\x02\x32\x32\x71\x18\x55\x02\x32\x33\x71\x19\x55\x02\x32\x34\x71\x1a\x55\x02\x32\x35\x71\x1b\x55\x02\x32\x36\x71\x1c\x55\x02\x32\x37\x71\x1d\x55\x02\x32\x38\x71\x1e\x55\x02\x32\x39\x71\x1f\x55\x02\x33\x30\x71\x20\x55\x02\x33\x31\x71\x21\x55\x02\x33\x32\x71\x22\x55\x02\x33\x33\x71\x23\x55\x02\x33\x34\x71\x24\x55\x02\x33\x35\x71\x25\x55\x02\x33\x36\x71\x26\x55\x02\x33\x37\x71\x27\x55\x02\x33\x38\x71\x28\x55\x02\x33\x39\x71\x29\x55\x02\x34\x30\x71\x2a\x55\x02\x34\x31\x71\x2b\x55\x02\x34\x32\x71\x2c\x55\x02\x34\x33\x71\x2d\x55\x02\x34\x34\x71\x2e\x55\x02\x34\x35\x71\x2f\x55\x02\x34\x36\x71\x30\x55\x02\x34\x37\x71\x31\x55\x02\x34\x38\x71\x32\x55\x02\x34\x39\x71\x33\x55\x02\x35\x30\x71\x34\x55\x02\x35\x31\x71\x35\x55\x02\x35\x32\x71\x36\x55\x02\x35\x33\x71\x37\x55\x02\x35\x34\x71\x38\x55\x02\x35\x35\x71\x39\x55\x02\x35\x36\x71\x3a\x55\x02\x35\x37\x71\x3b\x55\x02\x35\x38\x71\x3c\x55\x02\x35\x39\x71\x3d\x55\x02\x36\x30\x71\x3e\x55\x02\x36\x31\x71\x3f\x55\x02\x36\x32\x71\x40\x55\x02\x36\x33\x71\x41\x55\x02\x36\x34\x71\x42\x55\x02\x36\x35\x71\x43\x55\x02\x36\x36\x71\x44\x55\x02\x36\x37\x71\x45\x55\x02\x36\x38\x71\x46\x55\x02\x36\x39\x71\x47\x55\x02\x37\x30\x71\x48\x55\x02\x37\x31\x71\x49\x55\x02\x37\x32\x71\x4a\x55\x02\x37\x33\x71\x4b\x55\x02\x37\x34\x71\x4c\x55\x02\x37\x35\x71\x4d\x55\x02\x37\x36\x71\x4e\x55\x02\x37\x37\x71\x4f\x55\x02\x37\x38\x71\x50\x55\x02\x37\x39\x71\x51\x55\x02\x38\x30\x71\x52\x55\x02\x38\x31\x71\x53\x55\x02\x38\x32\x71\x54\x55\x02\x38\x33\x71\x55\x55\x02\x38\x34\x71\x56\x55\x02\x38\x35\x71\x57\x55\x02\x38\x36\x71\x58\x55\x02\x38\x37\x71\x59\x55\x02\x38\x38\x71\x5a\x55\x02\x38\x39\x71\x5b\x55\x02\x39\x30\x71\x5c\x55\x02\x39\x31\x71\x5d\x55\x02\x39\x32\x71\x5e\x55\x02\x39\x33\x71\x5f\x55\x02\x39\x34\x71\x60\x55\x02\x39\x35\x71\x61\x55\x02\x39\x36\x71\x62\x55\x02\x39\x37\x71\x63\x55\x02\x39\x38\x71\x64\x55\x02\x39\x39\x71\x65\x55\x03\x31\x30\x30\x71\x66\x55\x03\x31\x30\x31\x71\x67\x55\x03\x31\x30\x32\x71\x68\x55\x03\x31\x30\x33\x71\x69\x55\x03\x31\x30\x34\x71\x6a\x55\x03\x31\x30\x35\x71\x6b\x55\x03\x31\x30\x36\x71\x6c\x55\x03\x31\x30\x37\x71\x6d\x55\x03\x31\x30\x38\x71\x6e\x55\x03\x31\x30\x39\x71\x6f\x55\x03\x31\x31\x30\x71\x70\x55\x03\x31\x31\x31\x71\x71\x55\x03\x31\x31\x32\x71\x72\x55\x03\x31\x31\x33\x71\x73\x55\x03\x31\x31\x34\x71\x74\x55\x03\x31\x31\x35\x71\x75\x55\x03\x31\x31\x36\x71\x76\x55\x03\x31\x31\x37\x71\x77\x55\x03\x31\x31\x38\x71\x78\x55\x03\x31\x31\x39\x71\x79\x55\x03\x31\x32\x30\x71\x7a\x55\x03\x31\x32\x31\x71\x7b\x55\x03\x31\x32\x32\x71\x7c\x55\x03\x31\x32\x33\x71\x7d\x55\x03\x31\x32\x34\x71\x7e\x55\x03\x31\x32\x35\x71\x7f\x55\x03\x31\x32\x36\x71\x80\x55\x03\x31\x32\x37\x71\x81\x55\x03\x31\x32\x38\x71\x82\x55\x03\x31\x32\x39\x71\x83\x55\x03\x31\x33\x30\x71\x84\x55\x03\x31\x33\x31\x71\x85\x55\x03\x31\x33\x32\x71\x86\x55\x03\x31\x33\x33\x71\x87\x55\x03\x31\x33\x34\x71\x88\x55\x03\x31\x33\x35\x71\x89\x55\x03\x31\x33\x36\x71\x8a\x55\x03\x31\x33\x37\x71\x8b\x55\x03\x31\x33\x38\x71\x8c\x55\x03\x31\x33\x39\x71\x8d\x55\x03\x31\x34\x30\x71\x8e\x55\x03\x31\x34\x31\x71\x8f\x55\x03\x31\x34\x32\x71\x90\x55\x03\x31\x34\x33\x71\x91\x55\x03\x31\x34\x34\x71\x92\x55\x03\x31\x34\x35\x71\x93\x55\x03\x31\x34\x36\x71\x94\x55\x03\x31\x34\x37\x71\x95\x55\x03\x31\x34\x38\x71\x96\x55\x03\x31\x34\x39\x71\x97\x55\x03\x31\x35\x30\x71\x98\x55\x03\x31\x35\x31\x71\x99\x55\x03\x31\x35\x32\x71\x9a\x55\x03\x31\x35\x33\x71\x9b\x55\x03\x31\x35\x34\x71\x9c\x55\x03\x31\x35\x35\x71\x9d\x55\x03\x31\x35\x36\x71\x9e\x55\x03\x31\x35\x37\x71\x9f\x55\x03\x31\x35\x38\x71\xa0\x55\x03\x31\x35\x39\x71\xa1\x55\x03\x31\x36\x30\x71\xa2\x55\x03\x31\x36\x31\x71\xa3\x55\x03\x31\x36\x32\x71\xa4\x55\x03\x31\x36\x33\x71\xa5\x55\x03\x31\x36\x34\x71\xa6\x55\x03\x31\x36\x35\x71\xa7\x55\x03\x31\x36\x36\x71\xa8\x55\x03\x31\x36\x37\x71\xa9\x55\x03\x31\x36\x38\x71\xaa\x55\x03\x31\x36\x39\x71\xab\x55\x03\x31\x37\x30\x71\xac\x55\x03\x31\x37\x31\x71\xad\x55\x03\x31\x37\x32\x71\xae\x55\x03\x31\x37\x33\x71\xaf\x55\x03\x31\x37\x34\x71\xb0\x55\x03\x31\x37\x35\x71\xb1\x55\x03\x31\x37\x36\x71\xb2\x55\x03\x31\x37\x37\x71\xb3\x55\x03\x31\x37\x38\x71\xb4\x55\x03\x31\x37\x39\x71\xb5\x55\x03\x31\x38\x30\x71\xb6\x55\x03\x31\x38\x31\x71\xb7\x55\x03\x31\x38\x32\x71\xb8\x55\x03\x31\x38\x33\x71\xb9\x55\x03\x31\x38\x34\x71\xba\x55\x03\x31\x38\x35\x71\xbb\x55\x03\x31\x38\x36\x71\xbc\x55\x03\x31\x38\x37\x71\xbd\x55\x03\x31\x38\x38\x71\xbe\x55\x03\x31\x38\x39\x71\xbf\x55\x03\x31\x39\x30\x71\xc0\x55\x03\x31\x39\x31\x71\xc1\x55\x03\x31\x39\x32\x71\xc2\x55\x03\x31\x39\x33\x71\xc3\x55\x03\x31\x39\x34\x71\xc4\x55\x03\x31\x39\x35\x71\xc5\x55\x03\x31\x39\x36\x71\xc6\x55\x03\x31\x39\x37\x71\xc7\x55\x03\x31\x39\x38\x71\xc8\x55\x03\x31\x39\x39\x71\xc9\x55\x03\x32\x30\x30\x71\xca\x55\x03\x32\x30\x31\x71\xcb\x55\x03\x32\x30\x32\x71\xcc\x55\x03\x32\x30\x33\x71\xcd\x55\x03\x32\x30\x34\x71\xce\x55\x03\x32\x30\x35\x71\xcf\x55\x03\x32\x30\x36\x71\xd0\x55\x03\x32\x30\x37\x71\xd1\x55\x03\x32\x30\x38\x71\xd2\x55\x03\x32\x30\x39\x71\xd3\x55\x03\x32\x31\x30\x71\xd4\x55\x03\x32\x31\x31\x71\xd5\x55\x03\x32\x31\x32\x71\xd6\x55\x03\x32\x31\x33\x71\xd7\x55\x03\x32\x31\x34\x71\xd8\x55\x03\x32\x31\x35\x71\xd9\x55\x03\x32\x31\x36\x71\xda\x55\x03\x32\x31\x37\x71\xdb\x55\x03\x32\x31\x38\x71\xdc\x55\x03\x32\x31\x39\x71\xdd\x55\x03\x32\x32\x30\x71\xde\x55\x03\x32\x32\x31\x71\xdf\x55\x03\x32\x32\x32\x71\xe0\x55\x03\x32\x32\x33\x71\xe1\x55\x03\x32\x32\x34\x71\xe2\x55\x03\x32\x32\x35\x71\xe3\x55\x03\x32\x32\x36\x71\xe4\x55\x03\x32\x32\x37\x71\xe5\x55\x03\x32\x32\x38\x71\xe6\x55\x03\x32\x32\x39\x71\xe7\x55\x03\x32\x33\x30\x71\xe8\x55\x03\x32\x33\x31\x71\xe9\x55\x03\x32\x33\x32\x71\xea\x55\x03\x32\x33\x33\x71\xeb\x55\x03\x32\x33\x34\x71\xec\x55\x03\x32\x33\x35\x71\xed\x55\x03\x32\x33\x36\x71\xee\x55\x03\x32\x33\x37\x71\xef\x55\x03\x32\x33\x38\x71\xf0\x55\x03\x32\x33\x39\x71\xf1\x55\x03\x32\x34\x30\x71\xf2\x55\x03\x32\x34\x31\x71\xf3\x55\x03\x32\x34\x32\x71\xf4\x55\x03\x32\x34\x33\x71\xf5\x55\x03\x32\x34\x34\x71\xf6\x55\x03\x32\x34\x35\x71\xf7\x55\x03\x32\x34\x36\x71\xf8\x55\x03\x32\x34\x37\x71\xf9\x55\x03\x32\x34\x38\x71\xfa\x55\x03\x32\x34\x39\x71\xfb\x55\x03\x32\x35\x30\x71\xfc\x55\x03\x32\x35\x31\x71\xfd\x55\x03\x32\x35\x32\x71\xfe\x55\x03\x32\x35\x33\x71\xff\x55\x03\x32\x35\x34\x72\x00\x01\x00\x00\x55\x03\x32\x35\x35\x72\x01\x01\x00\x00\x65\x55\x04\x6d\x65\x6f\x77\x72\x02\x01\x00\x00\x4b\x05\x6a\x02\x01\x00\x00\x65\x2e"

	expect := make([]interface{}, 4)
	expect[1] = "meow"
	expect[2] = int64(5)
	expect[3] = "meow"

	countingList := make([]interface{}, 256)
	for i := 0; i != len(countingList); i++ {
		countingList[i] = fmt.Sprintf("%d", i)
	}
	expect[0] = countingList

	testList(t, input, expect)
}

func TestProtocol2Long(t *testing.T) {
	testBigInt(t, "\x80\x02\x8a\x00.", big.NewInt(0))
	testBigInt(t, "\x80\x02\x8a\x01\x01.", big.NewInt(1))
	testBigIntFromString(t, "\x80\x02\x8a\x0bR\xd3?\xd8\x9cY\xa5\xa7_\xc9\x04.", "5786663462362423463236434")
	testBigIntFromString(t, "\x80\x02\x8a\x0b\xae,\xc0'c\xa6ZX\xa06\xfb.", "-5786663462362423463236434")
	testBigInt(t, "\x80\x02\x8a\x01\xff.", big.NewInt(-1))

	testBigIntFromString(t, "\x80\x02\x8a\x11\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff.", "-340282366920938463463374607431768211456")

	testBigIntFromString(t, "\x80\x02\x8a\t\xff\xff\xff\xff\xff\xff\xff\xff\x00.", "18446744073709551615")
	testBigIntFromString(t, "\x80\x02\x8a\t\x01\x00\x00\x00\x00\x00\x00\x00\xff.", "-18446744073709551615")

	//tests the long4 opcode
	testBigIntFromString(t, "\x80\x02\x8b\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01.",
		"32317006071311007300714876688669951960444102669715484032130345427524655138867890893197201411522913463688717960921898019494119559150490921095088152386448283120630877367300996091750197750389652106796057638384067568276792218642619756161838094338476170470581645852036305042887575891541065808607552399123930385521914333389668342420684974786564569494856176035326322058077805659331026192708460314150258592864177116725943603718461857357598351152301645904403697613233287231227125684710820209725157101726931323469678542580656697935045997268352998638215525166389437335543602135433229604645318478604952148193555853611059596230656")

	testBigIntFromString(t, "\x80\x02\x8b\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff.",
		"-32317006071311007300714876688669951960444102669715484032130345427524655138867890893197201411522913463688717960921898019494119559150490921095088152386448283120630877367300996091750197750389652106796057638384067568276792218642619756161838094338476170470581645852036305042887575891541065808607552399123930385521914333389668342420684974786564569494856176035326322058077805659331026192708460314150258592864177116725943603718461857357598351152301645904403697613233287231227125684710820209725157101726931323469678542580656697935045997268352998638215525166389437335543602135433229604645318478604952148193555853611059596230656")

	testBigIntFromString(t, "\x80\x02\x8b\x01\x01\x00\x00*\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff.",
		"-32317006071311007300714876688669951960444102669715484032130345427524655138867890893197201411522913463688717960921898019494119559150490921095088152386448283120630877367300996091750197750389652106796057638384067568276792218642619756161838094338476170470581645852036305042887575891541065808607552399123930385521914333389668342420684974786564569494856176035326322058077805659331026192708460314150258592864177116725943603718461857357598351152301645904403697613233287231227125684710820209725157101726931323469678542580656697935045997268352998638215525166389437335543602135433229604645318478604952148193555853611059596230614")
}

func TestProtocol2TrueFalse(t *testing.T) {

	result, err := Bool(Unpickle(strings.NewReader("\x80\x02\x88.")))
	if err != nil {
		t.Fatal(err)
	}

	if result != true {
		t.Fatal("didnt get true")
	}

	result, err = Bool(Unpickle(strings.NewReader("\x80\x02\x89.")))
	if err != nil {
		t.Fatal(err)
	}

	if result != false {
		t.Fatal("didnt get false")
	}
}

func TestProtocol2Tuples(t *testing.T) {
	testList(t, "\x80\x02N\x85q\x00.", []interface{}{PickleNone{}})
	testList(t, "\x80\x02U\x05kittyq\x00K7\x86q\x01.", []interface{}{"kitty", int64(55)})
	testList(t, "\x80\x02U\x05kittyq\x00K7G@*\xbdp\xa3\xd7\n=\x87q\x01.", []interface{}{"kitty", int64(55), 13.37})

}
