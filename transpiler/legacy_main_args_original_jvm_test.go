package transpiler

import (
 "bytes"
 "context"
 "fmt"
 "go/ast"
 "go/parser"
 "go/token"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "testing"
 "time"
 "github.com/NickyBoy89/java2go/symbol"
)

// Maven discovery excludes unnamed packages. Exercise the same existing
// standalone producer directly, retaining the complete original Java.
func TestLegacyMainArgsOriginalJVMParity(t *testing.T) {
 source,err:=os.ReadFile("testdata/enclosing_qualifier_witness/Main.java");if err!=nil{t.Fatal(err)}
 root:=t.TempDir();input:=filepath.Join(root,"Main.java");generated:=filepath.Join(root,"generated")
 if err:=writeProjectFile(input,source);err!=nil{t.Fatal(err)}
 previous:=symbol.GlobalScope;t.Cleanup(func(){symbol.GlobalScope=previous})
 artifact:=os.Getenv("JAVA2GO_NORMALIZER_ARTIFACTS")
 t.Cleanup(func(){if artifact==""{return};err:=filepath.WalkDir(root,func(p string,d os.DirEntry,e error)error{if e!=nil||d.IsDir(){return e};if filepath.Ext(p)!=".go"&&filepath.Ext(p)!=".java"&&filepath.Base(p)!="go.mod"{return nil};rel,e:=filepath.Rel(root,p);if e!=nil{return e};raw,e:=os.ReadFile(p);if e!=nil{return e};return writeProjectFile(filepath.Join(artifact,"legacy",rel),raw)});if err!=nil{t.Error(err)}})
 var compilerOutput bytes.Buffer
 if err:=runInternal([]string{"-strict","-sync","-w","-output",generated,input},&compilerOutput,false);err!=nil{t.Fatalf("project producer: %v; output=%s",err,compilerOutput.Bytes())}
 pkg:=symbol.GlobalScope.FindPackage("");if pkg==nil{t.Fatal("missing unnamed Java package")}
 entry:=""
 for _,file:=range pkg.Files {if class:=file.FindClassScope("Main");class!=nil {for _,method:=range class.Methods {if projectMain(method){if entry!=""{t.Fatal("duplicate entry")};entry=symbol.GoIdentifier(method.Name)}}}}
 if entry==""{t.Fatal("missing actual public Java main entry")}
 parsed,err:=parser.ParseFile(token.NewFileSet(),filepath.Join(generated,"Main.go"),nil,0);if err!=nil{t.Fatal(err)}
 if parsed.Name.Name!="main"{t.Fatalf("unnamed package generated %q",parsed.Name.Name)}
 entryCount:=0
 for _,decl:=range parsed.Decls{fn,ok:=decl.(*ast.FuncDecl);if ok&&fn.Name.Name==entry{if fn.Recv!=nil||fn.Type.Params.NumFields()!=0{t.Fatal("legacy public main entry is not zero argument")};entryCount++}}
 if entryCount!=1{t.Fatal("actual public entry not unique")}
 // Add only the Go process entry; the unchanged generated Java main creates
 // its own canonical argument array through the repaired producer boundary.
 launcher:=fmt.Sprintf("package main\nfunc main(){%s()}\n",entry)
 if err:=writeProjectFile(filepath.Join(generated,"host_entry.go"),[]byte(launcher));err!=nil{t.Fatal(err)}
 repo,err:=filepath.Abs("..");if err!=nil{t.Fatal(err)}
 gomod:=fmt.Sprintf("module enclosing.generated\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\nreplace github.com/NickyBoy89/java2go => %s\n",strconv.Quote(filepath.ToSlash(repo)))
 if err:=writeProjectFile(filepath.Join(generated,"go.mod"),[]byte(gomod));err!=nil{t.Fatal(err)}
 run:=func(stage string,bound time.Duration,name string,args ...string)([]byte,[]byte){t.Helper();ctx,cancel:=context.WithTimeout(context.Background(),bound);defer cancel();cmd:=exec.CommandContext(ctx,name,args...);cmd.Dir=generated;cmd.Env=append(os.Environ(),"GOWORK=off");cmd.WaitDelay=5*time.Second;var out,errout bytes.Buffer;cmd.Stdout=&out;cmd.Stderr=&errout;e:=cmd.Run();if artifact!=""{if err:=writeProjectFile(filepath.Join(artifact,stage+".stdout"),out.Bytes());err!=nil{t.Fatal(err)};if err:=writeProjectFile(filepath.Join(artifact,stage+".stderr"),errout.Bytes());err!=nil{t.Fatal(err)}};if e!=nil||ctx.Err()!=nil{t.Fatalf("%s: %v timeout=%v stdout=%s stderr=%s",stage,e,ctx.Err(),out.Bytes(),errout.Bytes())};return out.Bytes(),errout.Bytes()}
 run("allpackage-race",5*time.Minute,"go","build","-race","-mod=mod","./...")
 binary:=filepath.Join(root,"enclosing-program");run("entry-race",5*time.Minute,"go","build","-race","-mod=mod","-o",binary,".")
 for _,mode:=range []string{"instance","static","priorlocal"}{for repeat:=1;repeat<=3;repeat++{t.Run(fmt.Sprintf("%s%d",mode,repeat),func(t *testing.T){want,err:=os.ReadFile("testdata/enclosing_qualifier_witness/"+mode+".stdout");if err!=nil{t.Fatal(err)};got,stderr:=run(fmt.Sprintf("%s%d",mode,repeat),time.Minute,binary,mode);if !bytes.Equal(got,want)||len(stderr)!=0{t.Fatalf("project stdout=%q stderr=%q differs from independently observed JVM stdout=%q",got,stderr,want)}})}}
}
