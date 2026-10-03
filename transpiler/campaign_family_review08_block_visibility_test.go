package transpiler

import "testing"

// All references are in one executable. Exiting each sibling block must restore
// the top-level Slot binding; the hoisted registry must retain neither local.
func TestCampaignFamilyReview08BlockVisibility(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>block-visibility</artifactId><version>1</version></project>`,
		"src/main/java/review/block/Main.java": `package review.block;
class Slot { public String label(){return "global";} }
public class Main {
 public static void main(String[] args){
  Slot initial=new Slot();System.out.println("before="+initial.label());
  {class Slot { public String label(){return "left";} }
   Slot left=new Slot();System.out.println("left="+left.label());}
  Slot between=new Slot();System.out.println("between="+between.label());
  {class Slot { public String label(){return "right";} }
   Slot right=new Slot();System.out.println("right="+right.label());}
  Slot after=new Slot();System.out.println("after="+after.label());
 }
}
`}, "review.block.Main", "before=global\nleft=left\nbetween=global\nright=right\nafter=global\n")
}
