package	input

import	(
	"os"
	"io"
	"../message"
	"gopkg.in/fsnotify.v1"

)



type	FileReader struct {
	GenericInput
	Source		string		`json:"source"`
	AppName		string		`json:"appname"`
	Priority	string		`json:"priority"`

	pos		int64
	prio		int
	watcher		*fsnotify.Watcher
}


func (file *FileReader)DriverName() string {
	return	"i_tailfile"
}


func (fr *FileReader) FileSize() (n int64) {
	fst,err	:= os.Stat(fr.Source)
	if err != nil {
		return 0
	}

	return fst.Size()
}




func (fr *FileReader) Read(p []byte) (n int, err error) {
	if fr.FileSize() == fr.pos {
		for {
			select {
				case	ev := <-fr.watcher.Events:
					if ev.Op&fsnotify.Write == fsnotify.Write {
						break
					}

				case	erw:= <-fr.watcher.Errors:
					return 0,erw
			}
		}
	}

	f,_	:= os.OpenFile( fr.Source, os.O_RDONLY, 0644 )
	defer	f.Close()

	fst,_	:= f.Stat()
	if fr.pos > fst.Size() {
		fr.pos = 0
	}
	f.Seek(fr.pos,0)

	n,err	= f.Read(p)
	fr.pos	+= int64(n)
	if err == io.EOF {
		err=nil
	}

	return
}


func (file *FileReader) Close() {
	file.watcher.Close()
}



func (file *FileReader)Run(dest chan<- Message, errchan chan<- error) {
	var err error
	file.end	= make(chan bool,1)

	file.prio,err	= message.PriorityDecode(file.Priority)
	if err != nil {
		errchan <- &InputError{ file.Driver, file.Id,"Priority "+file.Priority , err }
		return
	}

	file.watcher,err = fsnotify.NewWatcher()
	if err != nil {
		errchan <- &InputError{ file.Driver, file.Id,"Watcher "+file.Source , err }
		return
	}

	file.watcher.Add(file.Source)

	data	:= make(chan string)
	defer	file.Close()

	go reader_to_channel( file , data )

	for {
		select{
			case line := <- data:
				dest <- packmsg(file.Id, *message.CreateMessage(line, file.AppName, file.prio))

			case <- file.end:
				return
		}
	}
}
