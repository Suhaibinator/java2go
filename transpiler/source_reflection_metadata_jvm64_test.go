package transpiler

import (
 "fmt"
 "os"
 "path/filepath"
 "testing"
)

// Independently authored ordinary-Java controls keep actual upstream SerializedName
// unchanged. Expected bytes must be added only from the pinned JDK capture; each
// test independently reruns that JDK before strict all-package/entry race builds.
func sourceReflectionProjectObservations64(t *testing.T, project string) {
 t.Helper()
 root:=filepath.Join("testdata","source_reflection_metadata64",project)
 files:=map[string]string{}
 err:=filepath.WalkDir(root,func(path string,entry os.DirEntry,err error)error{
  if err!=nil||entry.IsDir(){return err}
  rel,err:=filepath.Rel(root,path);if err!=nil{return err}
  data,err:=os.ReadFile(path);if err!=nil{return err}
  files[filepath.ToSlash(rel)]=string(data);return nil
 })
 if err!=nil{t.Fatal(err)}
 var observations []campaignCompilerProjectObservation
 for _,seed:=range []int{17,41,97}{
  for repeat:=1;repeat<=3;repeat++{
   name:=fmt.Sprintf("seed-%d-repeat-%d",seed,repeat)
   oracleRoot:=filepath.Join("testdata","source_reflection_metadata64","oracles",project)
   stdout,err:=os.ReadFile(filepath.Join(oracleRoot,name+".stdout"));if err!=nil{t.Fatalf("independent actual JDK capture required: %v",err)}
   stderr,err:=os.ReadFile(filepath.Join(oracleRoot,name+".stderr"));if err!=nil{t.Fatalf("independent actual JDK stderr capture required: %v",err)}
   observations=append(observations,campaignCompilerProjectObservation{name:name,args:[]string{fmt.Sprint(seed)},stdout:string(stdout),stderr:string(stderr)})
  }
 }
 runCampaignCompilerStrictProjectObservations(t,files,"probe.app.Main",observations)
}

func TestSourceReflectionDeclaredFieldsJVM64(t *testing.T){sourceReflectionProjectObservations64(t,"fields")}
func TestSourceReflectionAnnotationValuesJVM64(t *testing.T){sourceReflectionProjectObservations64(t,"annotations")}
func TestSourceReflectionGenericSuperclassJVM64(t *testing.T){sourceReflectionProjectObservations64(t,"generic")}
func TestSourceReflectionDeclaredConstructorJVM64(t *testing.T){sourceReflectionProjectObservations64(t,"constructors")}
