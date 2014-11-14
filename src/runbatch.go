
package main

import (
  "flag"
  "bufio"
  "io"
  "os"
  "os/exec"
  "path/filepath"
  "fmt"
  "strconv"
  "strings"
  "time"
)

var resource = "-l gpu=GTX680"

func GetFiles(inputdir string) []string {
  files, _ := filepath.Glob(inputdir + "/*.gexf")
  return files
}

func GetOutName (outdir, file string, iterations int) string {
  outname := strings.TrimSuffix(file, filepath.Ext(file))
  absout, _ := filepath.Abs(outdir)
  return filepath.Join(absout, outname) + "-" + strconv.Itoa(iterations)
}

func WaitForNode(nodeId string) {
  nodeStatus := ""
  for nodeStatus != "r" {
    fmt.Println("Waiting for node", nodeId)
    command := exec.Command("preserve", "-list")
    output, _ := command.Output()
    outputString := string(output[:])
    lines := strings.Split(outputString, "\n")
    for _, line := range lines {
      if strings.Contains(line, "jdonkerv") {
        if strings.Fields(line)[0] == nodeId {
          nodeStatus = strings.Fields(line)[4]
          time.Sleep(1 * time.Second)
        }
      }
    }
  }
}

func ReserveNode() int {
  //"-native", resource, 
  command := exec.Command("preserve","-native", resource, 
  "-t", "02:00:00", "-#", "1")
  fmt.Println(command.Args)
  cmdout, _ := command.Output()

  nodeId := strings.Split(strings.Split(string(cmdout[:]), "\n")[0], " ")[2]
  nodeId = nodeId[:len(nodeId)-1]
  fmt.Println(nodeId)
  WaitForNode(nodeId)
  
  getId := exec.Command("preserve", "-list")
  out, _ := getId.Output()
  outputString := string(out[:])
  fmt.Println(outputString)
  lines := strings.Split(outputString, "\n")
  for _, line := range lines {
    if strings.Contains(line, "jdonkerv") {
      res, _ := strconv.Atoi(strings.Fields(line)[0])
      return res
    }
  }
  return -1
}

func CleanNode(nodeid int) {
  command := exec.Command("preserve", "-c", strconv.Itoa(nodeid))
  command.Run()
}

func main () {
  inputDirPtr := flag.String("indir", "", "The directory with the gexf files to run")
  outputDirPtr := flag.String("outdir", "", "The directory where the run time files will be placed.")
  
  flag.Parse()

  files := GetFiles(*inputDirPtr)

  for _, fin := range files {
    iterations := 100
    outfilename := GetOutName(*outputDirPtr, filepath.Base(fin), iterations)
    fmt.Println(outfilename)

    nodeid := ReserveNode()

    fmt.Println("Using node ", nodeid)
    command := exec.Command("prun", "-no-panda", "-reserve", strconv.Itoa(nodeid),
    "-native", resource, "./ap", "1", "-i", fin,
    "-n", strconv.Itoa(iterations))

    grep := exec.Command("grep", "time", "-A", "1")
    commandOut, _ := command.StdoutPipe()
    grep.Stdin = commandOut

    outfile, _ := os.Create(outfilename)
    defer outfile.Close()

    writer := bufio.NewWriter(outfile)
    defer writer.Flush()

    grepOut, _ := grep.StdoutPipe()
    grep.Start()
    command.Start()
    io.Copy(writer, grepOut)
    grep.Wait()
    command.Wait()

    CleanNode(nodeid)
  }
}
