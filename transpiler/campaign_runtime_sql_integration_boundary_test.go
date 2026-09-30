package transpiler

import "testing"

func TestCampaignRuntimeSQLBoundedDateStorageIntegration(t *testing.T) {
	const source = `import java.sql.Date;import java.util.function.Supplier;
class BoundedDateCell<T extends java.util.Date> {
 T value;
 BoundedDateCell(T value){this.value=value;}
 T get(){return value;}
 void set(T value){this.value=value;}
}
public class CampaignSQLBoundedStorage {
 static <T extends java.util.Date> T identity(T value){return value;}
 @SuppressWarnings({"rawtypes","unchecked"}) public static String run(){
  Date original=new Date(7L);BoundedDateCell<Date> typed=new BoundedDateCell<Date>(original);BoundedDateCell raw=typed;
  raw.set(new java.sql.Time(9L));Object broad=raw.get();java.util.Date base=typed.get();boolean late=false;
  try{Date wrong=typed.get();wrong.getTime();}catch(ClassCastException expected){late=true;}
  typed.set(original);Supplier<Date> supplier=typed::get;Date returned=supplier.get();boolean same=returned==original;
  typed.set(null);boolean absent=typed.get()==null;Date nullable=identity((Date)null);
  return broad.getClass().getName()+":"+base.getTime()+":"+late+":"+same+":"+absent+":"+(nullable==null)+":"+identity(original).getTime();
 }
}`
	campaignDateTimeOracle(t, "CampaignSQLBoundedStorage", source)
}

func TestCampaignRuntimeSQLCalendarReturnOwnerIntegration(t *testing.T) {
	const source = `import java.sql.Date;import java.util.Calendar;import java.util.GregorianCalendar;import java.util.TimeZone;import java.util.function.Supplier;
public class CampaignSQLCalendarOwner {
 static String owner(java.util.Date value){return "util:"+value.getTime();}
 static String owner(Date value){return "sql:"+value.getTime();}
 public static String run(){
  Calendar calendar=new GregorianCalendar(TimeZone.getTimeZone("UTC"));calendar.setTimeInMillis(7L);
  String direct=owner(calendar.getTime());String chained=calendar.getTime().getClass().getName();
  Supplier<java.util.Date> supplier=calendar::getTime;java.util.Date via=supplier.get();
  return direct+":"+chained+":"+owner(via)+":"+calendar.getTime().compareTo(new java.util.Date(7L));
 }
}`
	campaignDateTimeOracle(t, "CampaignSQLCalendarOwner", source)
}

func TestCampaignRuntimeBoundedOwnerMethodReferenceConsumers(t *testing.T) {
	const source = `import java.sql.Date;import java.util.function.Supplier;
interface LongDateRead {long get();}
class NumberMethodReferenceCell<T extends Number>{T value;NumberMethodReferenceCell(T value){this.value=value;}T get(){return value;}}
class DateMethodReferenceCell<T extends java.util.Date>{T value;DateMethodReferenceCell(T value){this.value=value;}T get(){return value;}void set(T value){this.value=value;}}
public class CampaignBoundedReferenceConsumers {
 @SuppressWarnings({"rawtypes","unchecked"}) public static String run(){
  NumberMethodReferenceCell<Integer> numbers=new NumberMethodReferenceCell<Integer>(17);LongDateRead widened=numbers::get;
  DateMethodReferenceCell<Date> dates=new DateMethodReferenceCell<Date>(new Date(3L));Supplier<Date> narrow=dates::get;Supplier<java.util.Date> broad=dates::get;Supplier<Object> object=dates::get;Runnable discarded=dates::get;
  DateMethodReferenceCell raw=dates;raw.set(new java.sql.Time(9L));discarded.run();boolean late=false;int sideEffects=0;try{Date result=narrow.get();sideEffects++;result.getTime();}catch(ClassCastException expected){late=true;}
  java.util.Date observed=broad.get();Object erased=object.get();raw.set(null);return widened.get()+":"+observed.getClass().getName()+":"+(erased==observed)+":"+late+":"+sideEffects+":"+(narrow.get()==null);
 }
}`
	campaignDateTimeOracle(t, "CampaignBoundedReferenceConsumers", source)
}

func TestCampaignRuntimeCanonicalOwnerInferenceEquality(t *testing.T) {
	for _, test := range []struct {
		name, source, left, right string
		same                      bool
	}{
		{"sql import", "import java.sql.Date; class Owner {}", "Date", "java.sql.Date", true},
		{"sql import excludes util", "import java.sql.Date; class Owner {}", "java.util.Date", "Date", false},
		{"util import", "import java.util.Date; class Owner {}", "Date", "java.util.Date", true},
		{"util import excludes sql", "import java.util.Date; class Owner {}", "java.sql.Date", "Date", false},
		{"wildcard", "import java.sql.*; class Owner {}", "Date", "java.util.Date", false},
		{"array component", "import java.sql.Date; class Owner {}", "Date[]", "java.util.Date[]", false},
		{"source shadows", "class Date {}", "Date", "java.util.Date", false},
		{"source self", "class Date {}", "Date", "Date", true},
		{"binder shadows", "class Owner<Date> {}", "Date", "java.util.Date", false},
		{"binder self", "class Owner<Date> {}", "Date", "Date", true},
		{"unrelated qualified", "class Owner {}", "vendor.Date", "java.sql.Date", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			for _, pair := range [][2]string{{test.left, test.right}, {test.right, test.left}} {
				if got := javaInferenceSameType(pair[0], pair[1], helper.Ctx); got != test.same {
					t.Fatalf("sameType(%q,%q)=%v, want %v", pair[0], pair[1], got, test.same)
				}
			}
		})
	}
}
