package transpiler

import "testing"

func TestCampaignRuntimeSQLDateIdentityAndCalendar(t *testing.T) {
	const source = `import java.util.Calendar;import java.util.GregorianCalendar;import java.util.TimeZone;
 public class CampaignSQLIdentity {
  static String identity(java.util.Date date){Object value=date;return value.getClass().getName()+":"+(value instanceof java.util.Date)+":"+date.getTime();}
  public static String run(){java.sql.Date date=new java.sql.Date(-1L);java.sql.Time time=new java.sql.Time(42L);java.sql.Timestamp stamp=new java.sql.Timestamp(1234L);Calendar c=new GregorianCalendar(TimeZone.getTimeZone("UTC"));c.setTime(stamp);return identity(date)+":"+identity(time)+":"+identity(stamp)+":"+c.getTimeInMillis()+":"+(date==((Object)date));}
 }`
	campaignDateTimeOracle(t, "CampaignSQLIdentity", source)
}

func TestCampaignRuntimeSQLTimestampPrecision(t *testing.T) {
	const source = `import java.sql.Timestamp;import java.util.Date;
 public class CampaignSQLPrecision {
  public static String run(){Timestamp stamp=new Timestamp(-1L);long negative=stamp.getTime();int nanos=stamp.getNanos();stamp.setNanos(123456789);long adjusted=stamp.getTime();Timestamp same=new Timestamp(adjusted);Date plain=new Date(adjusted);boolean utilEqual=plain.equals(stamp);boolean stampEqual=stamp.equals(plain);int order=stamp.compareTo(same);stamp.setTime(-1001L);long reset=stamp.getTime();int resetNanos=stamp.getNanos();String bad="";try{stamp.setNanos(1000000000);}catch(IllegalArgumentException failure){bad=failure.getMessage();}Date polymorphic=stamp;polymorphic.setTime(1500L);return negative+":"+nanos+":"+adjusted+":"+utilEqual+":"+stampEqual+":"+order+":"+reset+":"+resetNanos+":"+bad+":"+polymorphic.getTime()+":"+stamp.getNanos()+":"+(plain.hashCode()==same.hashCode());}
 }`
	campaignDateTimeOracle(t, "CampaignSQLPrecision", source)
}

func TestCampaignRuntimeSQLDateTimeText(t *testing.T) {
	const source = `import java.util.TimeZone;import java.sql.Date;import java.sql.Time;import java.sql.Timestamp;
 public class CampaignSQLText {
  public static String run(){TimeZone previous=TimeZone.getDefault();TimeZone.setDefault(TimeZone.getTimeZone("UTC"));Date date=Date.valueOf("2023-2-29");Time time=Time.valueOf("1:2:3");Timestamp stamp=Timestamp.valueOf("2024-02-29 12:34:56.123456789");String utc=date.toString()+":"+time.toString()+":"+stamp.toString()+":"+stamp.getNanos();TimeZone.setDefault(TimeZone.getTimeZone("GMT+05:45"));String zone=new Date(-1L).toString()+":"+new Time(-1L).toString();TimeZone.setDefault(previous);return utc+":"+zone;}
 }`
	campaignDateTimeOracle(t, "CampaignSQLText", source)
}

func TestCampaignRuntimeSQLTextBoundaries(t *testing.T) {
	const source = `import java.util.TimeZone;import java.sql.Date;import java.sql.Time;import java.sql.Timestamp;
 public class CampaignSQLTextBounds {
  static String parse(int kind,String text){try{if(kind==0)return Date.valueOf(text).toString();if(kind==1)return Time.valueOf(text).toString();return Timestamp.valueOf(text).toString();}catch(IllegalArgumentException failure){return failure.getClass().getSimpleName()+":"+failure.getMessage();}}
  public static String run(){TimeZone previous=TimeZone.getDefault();TimeZone.setDefault(TimeZone.getTimeZone("UTC"));String result=parse(0,null)+"|"+parse(0,"2023-2-30")+"|"+parse(0,"2023-13-01")+"|"+parse(0,"2023-0x-01")+"|"+parse(1,"25:61:61")+"|"+parse(1,"x:02:03")+"|"+parse(2," 2024-02-29 23:59:60.00100 ")+"|"+parse(2,"2024-02-29 00:00:00.1234567890")+"|"+parse(2,null);TimeZone.setDefault(previous);return result;}
 }`
	campaignDateTimeOracle(t, "CampaignSQLTextBounds", source)
}

func TestCampaignRuntimeSQLNullAndNumericConversions(t *testing.T) {
	const source = `import java.sql.Date;import java.sql.Time;import java.sql.Timestamp;
 public class CampaignSQLConversions {
  public static String run(){Integer input=17;Timestamp stamp=new Timestamp(input);Integer update=1234;java.util.Date base;base=stamp;base.setTime(update);Byte fraction=7;stamp.setNanos(fraction);boolean boxedNull=false;Long empty=null;try{new Timestamp(empty);}catch(NullPointerException failure){boxedNull=true;}Date date=null;Time time=null;boolean dateNull=false;boolean timeNull=false;try{date.getTime();}catch(NullPointerException failure){dateNull=true;}try{time.hashCode();}catch(NullPointerException failure){timeNull=true;}String text=null;boolean textNull=false;try{Date.valueOf(text);}catch(IllegalArgumentException failure){textNull=true;}java.util.Date[] values=new java.util.Date[]{stamp};return base.getTime()+":"+stamp.getNanos()+":"+boxedNull+":"+dateNull+":"+timeNull+":"+textNull+":"+(values[0]==stamp)+":"+values[0].getClass().getName();}
 }`
	campaignDateTimeOracle(t, "CampaignSQLConversions", source)
}

func TestCampaignRuntimeDateOwnerImportlessDefaultStable(t *testing.T) {
	for i := 0; i < 1000; i++ {
		owner, ok := canonicalIntrinsicOwner("Date", Ctx{})
		if !ok || owner != "java.util.Date" {
			t.Fatalf("resolution %d: %q, %v", i, owner, ok)
		}
	}
}

func TestCampaignRuntimeSQLReverseImportCastsAndArrays(t *testing.T) {
	const source = `import java.util.Date;
 public class CampaignSQLReverse {
  public static String run(){java.sql.Date value;value=new java.sql.Date(3L);Date base;base=value;Object object=base;java.sql.Date cast=(java.sql.Date)object;Date back=(java.util.Date)object;java.sql.Date[] dates=new java.sql.Date[]{cast};Object[] storage=dates;boolean rejected=false;try{storage[0]=new Date(3L);}catch(ArrayStoreException failure){rejected=true;}java.sql.Date empty=(java.sql.Date)null;return value.getClass().getName()+":"+dates[0].getClass().getName()+":"+(back==value)+":"+rejected+":"+(empty==null);}
 }`
	campaignDateTimeOracle(t, "CampaignSQLReverse", source)
}

func TestCampaignRuntimeSQLParsingErrorOrder(t *testing.T) {
	const source = `import java.sql.Time;import java.sql.Timestamp;
 public class CampaignSQLErrorOrder {
  static String parse(boolean time,String text){try{return time?Time.valueOf(text).toString():Timestamp.valueOf(text).toString();}catch(IllegalArgumentException failure){return failure.getClass().getSimpleName()+":"+failure.getMessage();}}
  public static String run(){return parse(true,"1::2")+"|"+parse(false,"2024-01-01 :02:03")+"|"+parse(false,"2024-01-01 01:02:x.");}
 }`
	campaignDateTimeOracle(t, "CampaignSQLErrorOrder", source)
}

func TestCampaignRuntimeSQLDateProtocolNullReceiver(t *testing.T) {
	const source = `import java.util.Date;
 public class CampaignSQLNullProtocol {
  static int calls=0;
  static long argument(){calls++;return 7L;}
  static Object object(){calls++;return null;}
  static Date date(){calls++;return null;}
  public static String run(){Date value=null;int caught=0;try{value.getTime();}catch(NullPointerException e){caught++;}try{value.setTime(argument());}catch(NullPointerException e){caught++;}try{value.equals(object());}catch(NullPointerException e){caught++;}try{value.hashCode();}catch(NullPointerException e){caught++;}try{value.compareTo(date());}catch(NullPointerException e){caught++;}return caught+":"+calls;}
 }`
	campaignDateTimeOracle(t, "CampaignSQLNullProtocol", source)
}
