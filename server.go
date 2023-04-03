// Package mbserver implments a Modbus server (slave).
package mbserver

import (
	"io"
	"net"
    "math/rand"
    "time"
    "encoding/csv" //added
    "fmt"
    "os"
    "log"
    "strconv"   //added

	"github.com/goburrow/serial"
)

// Server is a Modbus slave with allocated memory for discrete inputs, coils, etc.
type Server struct {
	// Debug enables more verbose messaging.
	Debug            bool
	listeners        []net.Listener
	ports            []serial.Port
	requestChan      chan *Request
	function         [256](func(*Server, Framer) ([]byte, *Exception))
	DiscreteInputs   []byte
	Coils            []byte
	HoldingRegisters []uint16
	InputRegisters   []uint16
}

// Request contains the connection and Modbus frame.
type Request struct {
	conn  io.ReadWriteCloser
	frame Framer
}

// NewServer creates a new Modbus server (slave).
func NewServer() *Server {
	s := &Server{}

	// Allocate Modbus memory maps.
	s.DiscreteInputs = make([]byte, 65536)
	s.Coils = make([]byte, 65536)
	s.HoldingRegisters = make([]uint16, 65536)
	s.InputRegisters = make([]uint16, 65536)

	// Add default functions.
	s.function[1] = ReadCoils
	s.function[2] = ReadDiscreteInputs
	s.function[3] = ReadHoldingRegisters
	s.function[4] = ReadInputRegisters
	s.function[5] = WriteSingleCoil
	s.function[6] = WriteHoldingRegister
	s.function[15] = WriteMultipleCoils
	s.function[16] = WriteHoldingRegisters

	s.requestChan = make(chan *Request)
	go s.handler()

	rand.Seed(time.Now().UnixNano())
	go s.random_generator()

	return s
}

// RegisterFunctionHandler override the default behavior for a given Modbus function.
func (s *Server) RegisterFunctionHandler(funcCode uint8, function func(*Server, Framer) ([]byte, *Exception)) {
	s.function[funcCode] = function
}

func (s *Server) handle(request *Request) Framer {
	var exception *Exception
	var data []byte

	response := request.frame.Copy()

	function := request.frame.GetFunction()
	if s.function[function] != nil {
		data, exception = s.function[function](s, request.frame)
		response.SetData(data)
	} else {
		exception = &IllegalFunction
	}

	if exception != &Success {
		response.SetException(exception)
	}

	return response
}

func (s *Server) random_generator() {
	// test change.
	min := 120
	max := 150

	fd, error := os.Open("data.csv")

	fmt.Println("Successfully opened the CSV file")
	defer fd.Close()

	if error != nil {
		fmt.Println(error)
	}

	s.HoldingRegisters[6337] = 100
	s.HoldingRegisters[6339] = 48532
	s.HoldingRegisters[6341] = 17179
	s.HoldingRegisters[6343] = 16477
	s.HoldingRegisters[6345] = 15861
	s.HoldingRegisters[6347] = 15821
	s.HoldingRegisters[6349] = 17178
	s.HoldingRegisters[6351] = 16078

	csvReader := csv.NewReader(fd)
	rec, err := csvReader.Read()
	if err != nil {
		log.Fatal(err)
	}
	

	for i:=6337;i<6345 ;i++ {
		ui64, err := strconv.ParseUint(rec[1+i], 10, 64)
		if err != nil {
      			panic(err)
  		}
		ui := uint16(ui64)
		s.HoldingRegisters[6337+i] = ui
		fmt.Printf("%+v\n", s.HoldingRegisters[6337+i])
	}
	
	for i := 6346; i < 6353; i+=2 {
		s.HoldingRegisters[i] = uint16(rand.Intn(max - min + 1) + min)
	}
	time.Sleep(time.Second * time.Duration(rand.Intn(10)))	

}

// All requests are handled synchronously to prevent modbus memory corruption.
func (s *Server) handler() {
	for {
		request := <-s.requestChan
		response := s.handle(request)
		request.conn.Write(response.Bytes())
	}
}

// Close stops listening to TCP/IP ports and closes serial ports.
func (s *Server) Close() {
	for _, listen := range s.listeners {
		listen.Close()
	}
	for _, port := range s.ports {
		port.Close()
	}
}
