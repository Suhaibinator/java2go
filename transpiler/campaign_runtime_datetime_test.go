package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func campaignDateTimeOracle(t *testing.T, name, source string) {
	t.Helper()
	want := campaignRuntimeJavaOracle(t, name, source)
	t.Logf("JVM datetime oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestDateTime(t *testing.T){if got:=Run();got==nil||!slices.Equal(got.UTF16Copy(),%#v){t.Fatalf("JVM %%q != Go JavaString %%#v",%q,got)}}`, utf16.Encode([]rune(want)), want))
}

func TestCampaignRuntimeDateAndParsePosition(t *testing.T) {
	const source = `import java.util.Date;import java.text.*;
public class CampaignDateValues {
 public static String run(){
  Date negative=new Date(-1L);Date same=new Date(-1L);Date changed=new Date(0L);changed.setTime(4294967296L);
  ParsePosition position=new ParsePosition(-3);int initial=position.getErrorIndex();position.setIndex(7);position.setErrorIndex(4);
  ParseException failure=new ParseException("invalid",9);failure.initCause(new IllegalArgumentException("cause"));int offset=0;try{throw failure;}catch(ParseException caught){offset=caught.getErrorOffset();}
  return negative.getTime()+":"+negative.equals(same)+":"+negative.hashCode()+":"+changed.getTime()+":"+changed.hashCode()+":"+initial+":"+position.getIndex()+":"+position.getErrorIndex()+":"+offset+":"+failure.getMessage()+":"+failure.getCause().getMessage();
 }
}`
	campaignDateTimeOracle(t, "CampaignDateValues", source)
}

func TestCampaignRuntimeTimeZone(t *testing.T) {
	const source = `import java.util.*;
public class CampaignTimeZones {
 public static String run(){
  TimeZone utc=TimeZone.getTimeZone("UTC");TimeZone custom=TimeZone.getTimeZone("GMT+0545");TimeZone invalid=TimeZone.getTimeZone("GMT+25:00");
  TimeZone winter=TimeZone.getTimeZone("America/New_York");
  TimeZone original=TimeZone.getDefault();TimeZone.setDefault(custom);TimeZone copy=TimeZone.getDefault();copy.setID("changed");String saved=TimeZone.getDefault().getID();TimeZone.setDefault(original);
  boolean nullZone=false;try{TimeZone.getTimeZone((String)null);}catch(NullPointerException expected){nullZone=true;}
  return utc.getID()+":"+utc.getRawOffset()+":"+custom.getID()+":"+custom.getOffset(-1L)+":"+invalid.getID()+":"+winter.getRawOffset()+":"+winter.getOffset(1704067200000L)+":"+winter.getOffset(1719792000000L)+":"+saved+":"+nullZone;
 }
}`
	campaignDateTimeOracle(t, "CampaignTimeZones", source)
}

func TestCampaignRuntimeThrowableInitCause(t *testing.T) {
	const source = `class CustomParseFailure extends Exception{CustomParseFailure(String message){super(message);}}
public class CampaignInitCause {
 public static String run(){
  Exception fresh=new Exception("fresh");Throwable cause=new IllegalArgumentException("cause");Throwable result=fresh.initCause(cause);String repeated="";try{fresh.initCause(null);}catch(IllegalStateException expected){repeated=expected.getMessage();}
  Exception self=new Exception("self");String selfMessage="";try{self.initCause(self);}catch(IllegalArgumentException expected){selfMessage=expected.getMessage();}
  Exception explicit=new Exception("explicit",null);boolean blocked=false;try{explicit.initCause(cause);}catch(IllegalStateException expected){blocked=true;}
  Exception nullable=new Exception((String)null);nullable.initCause(null);boolean second=false;try{nullable.initCause(null);}catch(IllegalStateException expected){second=true;}
  CustomParseFailure custom=new CustomParseFailure("custom");custom.initCause(cause);
  return (result==fresh)+":"+(fresh.getCause()==cause)+":"+repeated+":"+selfMessage+":"+blocked+":"+second+":"+(custom.getCause()==cause);
 }
}`
	campaignDateTimeOracle(t, "CampaignInitCause", source)
}

func TestCampaignRuntimeGregorianCalendar(t *testing.T) {
	const source = `import java.util.*;
public class CampaignCalendars {
 static long instant(String zone,boolean lenient,int year,int month,int day,int hour,int minute){Calendar c=new GregorianCalendar(TimeZone.getTimeZone(zone));c.clear();c.setLenient(lenient);c.set(year,month,day,hour,minute,0);return c.getTimeInMillis();}
 static String invalid(int year,int month,int day,int hour,int minute){try{return "accepted:"+instant("UTC",false,year,month,day,hour,minute);}catch(IllegalArgumentException failure){return failure.getMessage();}}
 public static String run(){
  TimeZone original=TimeZone.getDefault();TimeZone.setDefault(TimeZone.getTimeZone("UTC"));
  Calendar calendar=new GregorianCalendar(2024,1,29,12,34,56);calendar.set(Calendar.MILLISECOND,123);Date snapshot=calendar.getTime();calendar.set(Calendar.SECOND,57);long before=snapshot.getTime();snapshot.setTime(0L);long changed=calendar.getTimeInMillis();
  calendar.setTime(new Date(-1L));String fields=calendar.get(Calendar.YEAR)+":"+calendar.get(Calendar.MONTH)+":"+calendar.get(Calendar.DAY_OF_MONTH)+":"+calendar.get(Calendar.HOUR_OF_DAY)+":"+calendar.get(Calendar.MINUTE)+":"+calendar.get(Calendar.SECOND)+":"+calendar.get(Calendar.MILLISECOND);
  long lenient=instant("UTC",true,2023,1,29,25,0);long old=instant("UTC",false,1582,9,4,0,0);long modern=instant("UTC",false,1582,9,15,0,0);
  String gap="";try{instant("America/New_York",false,2024,2,10,2,30);}catch(IllegalArgumentException failure){gap=failure.getMessage();}
  long overlap=instant("America/New_York",false,2024,10,3,1,30);long normalized=instant("America/New_York",true,2024,2,10,2,30);
  TimeZone.setDefault(original);
  return before+":"+changed+":"+fields+":"+lenient+":"+invalid(2023,1,29,0,0)+":"+invalid(2024,12,1,0,0)+":"+invalid(2024,0,1,24,0)+":"+old+":"+(modern-old)+":"+invalid(1582,9,10,0,0)+":"+gap+":"+overlap+":"+normalized+":"+(calendar instanceof GregorianCalendar);
 }
}`
	campaignDateTimeOracle(t, "CampaignCalendars", source)
}

func TestCampaignRuntimeCalendarStateAndEra(t *testing.T) {
	const source = `import java.util.*;
public class CampaignCalendarState {
 public static String run(){
  TimeZone previous=TimeZone.getDefault();TimeZone.setDefault(TimeZone.getTimeZone("UTC"));
  Calendar c=new GregorianCalendar(2024,0,2);String midnight=c.get(Calendar.HOUR_OF_DAY)+":"+c.get(Calendar.MILLISECOND);
  c.clear();c.set(Calendar.ERA,0);c.set(Calendar.YEAR,1);c.set(Calendar.MONTH,0);c.set(Calendar.DAY_OF_MONTH,1);long bc=c.getTimeInMillis();String era=c.get(Calendar.ERA)+":"+c.get(Calendar.YEAR)+":"+c.get(Calendar.DAY_OF_MONTH);
  c.clear();c.setLenient(false);c.set(1500,1,29);long julian=c.getTimeInMillis();c.set(1700,1,29);String leap="";try{c.getTimeInMillis();}catch(IllegalArgumentException bad){leap=bad.getMessage();}
  c.clear();c.set(2024,0,1);c.set(Calendar.HOUR_OF_DAY,13);c.set(Calendar.HOUR,2);c.set(Calendar.AM_PM,0);String conflict="";try{c.getTimeInMillis();}catch(IllegalArgumentException bad){conflict=bad.getMessage();}c.setLenient(true);int early=c.get(Calendar.HOUR_OF_DAY);c.set(Calendar.HOUR_OF_DAY,22);int late=c.get(Calendar.HOUR_OF_DAY);c.set(Calendar.AM_PM,0);int reset=c.get(Calendar.HOUR_OF_DAY);
  Object calendarClass=Calendar.class;TimeZone.setDefault(previous);return midnight+":"+bc+":"+era+":"+julian+":"+leap+":"+conflict+":"+early+":"+late+":"+reset+":"+(calendarClass!=GregorianCalendar.class);
 }
}`
	campaignDateTimeOracle(t, "CampaignCalendarState", source)
}

func TestCampaignRuntimeTimeZoneHistorical(t *testing.T) {
	const source = `import java.util.TimeZone;
public class CampaignHistoricalZones {
 public static String run(){return TimeZone.getTimeZone("GMT0").getID()+":"+TimeZone.getTimeZone("GMT-0").getID()+":"+TimeZone.getTimeZone("America/New_York").getOffset(-5364662400000L)+":"+TimeZone.getTimeZone("Asia/Kolkata").getOffset(-5364662400000L)+":"+TimeZone.getTimeZone("GMT+001").getID();}
}`
	campaignDateTimeOracle(t, "CampaignHistoricalZones", source)
}

func TestCampaignRuntimeThrowableInitCauseOverride(t *testing.T) {
	const source = `class CauseOverride extends Exception {
 int calls;
 CauseOverride(){super("override");}
 public synchronized CauseOverride initCause(Throwable cause){calls++;super.initCause(cause);return this;}
}
class RejectCause extends Exception {
 public RejectCause initCause(Throwable cause){throw new IllegalArgumentException("override rejected");}
}
public class CampaignInitCauseOverride {
 public static String run(){CauseOverride custom=new CauseOverride();Exception base=custom;Exception cause=new Exception("cause");Throwable same=base.initCause(cause);String rejected="";Exception reject=new RejectCause();try{reject.initCause(cause);}catch(IllegalArgumentException failure){rejected=failure.getMessage();}return custom.calls+":"+(same==custom)+":"+(base.getCause()==cause)+":"+rejected;}
}`
	campaignDateTimeOracle(t, "CampaignInitCauseOverride", source)
}

func TestCampaignRuntimeThrowableInitCauseOverload(t *testing.T) {
	const source = `class CauseOverload extends Exception {
 int calls;
 public CauseOverload initCause(Exception cause){calls++;return this;}
}
public class CampaignInitCauseOverload {
 public static String run(){CauseOverload value=new CauseOverload();Exception base=value;base.initCause((Throwable)null);int before=value.calls;value.initCause(new Exception("local"));return before+":"+value.calls+":"+(base.getCause()==null);}
}`
	campaignDateTimeOracle(t, "CampaignInitCauseOverload", source)
}

func TestCampaignRuntimeThrowableInitCauseInherited(t *testing.T) {
	const source = `class InheritedCauseBase extends Exception {
  int calls;
  public InheritedCauseBase initCause(Throwable cause){calls++;super.initCause(cause);return this;}
 }
 class InheritedCauseChild extends InheritedCauseBase {
  int overloads;
  public InheritedCauseChild initCause(Exception cause){overloads++;return this;}
 }
 class ImplicitCauseChild extends Exception { ImplicitCauseChild(int ignored){} }
 public class CampaignInitCauseInherited {
  public static String run(){InheritedCauseChild child=new InheritedCauseChild();Exception base=child;Throwable result=base.initCause((Throwable)null);child.initCause(new Exception());ImplicitCauseChild implicit=new ImplicitCauseChild(1);implicit.initCause(new Exception("ok"));return child.calls+":"+child.overloads+":"+(result==child)+":"+(base.getCause()==null)+":"+implicit.getCause().getMessage();}
 }`
	campaignDateTimeOracle(t, "CampaignInitCauseInherited", source)
}

func TestCampaignRuntimeThrowableInitCauseMonitor(t *testing.T) {
	const source = `class MonitorCause extends Exception {
  static Exception target;
  public String toString(){boolean held=Thread.holdsLock(target);synchronized(target){return "held="+held+":"+Thread.holdsLock(target);}}
 }
 public class CampaignInitCauseMonitor {
  public static String run(){Exception target=new Exception();MonitorCause.target=target;target.initCause(null);String message="";try{target.initCause(new MonitorCause());}catch(IllegalStateException failure){message=failure.getMessage();}return message+":"+Thread.holdsLock(target);}
 }`
	campaignDateTimeOracle(t, "CampaignInitCauseMonitor", source)
}

func TestCampaignRuntimeThrowableInitCauseInheritedMonitor(t *testing.T) {
	const source = `class ReturnedCauseBase extends Exception {
  public ReturnedCauseBase initCause(Throwable cause){super.initCause(cause);return this;}
 }
 class ReturnedCauseChild extends ReturnedCauseBase {}
 public class CampaignInitCauseInheritedMonitor {
  public static String run(){ReturnedCauseChild child=new ReturnedCauseChild();ReturnedCauseBase base=child;Throwable result;boolean held;synchronized(child){result=base.initCause(null);held=Thread.holdsLock(result);}return (result==child)+":"+(result==base)+":"+held+":"+Thread.holdsLock(result);}
 }`
	campaignDateTimeOracle(t, "CampaignInitCauseInheritedMonitor", source)
}

func TestCampaignRuntimeThrowableInitCauseCovariant(t *testing.T) {
	const source = `class CovariantCause extends Exception {
  public CovariantCause initCause(Throwable cause){super.initCause(cause);return this;}
 }
 class NullCauseResult extends Exception {
  public NullCauseResult initCause(Throwable cause){return null;}
 }
 public class CampaignInitCauseCovariant {
  public static String run(){CovariantCause original=new CovariantCause();CovariantCause result;result=original.initCause(null);NullCauseResult empty=new NullCauseResult();Exception erased=empty;return (result==original)+":"+(empty.initCause(null)==null)+":"+(erased.initCause(null)==null);}
 }`
	campaignDateTimeOracle(t, "CampaignInitCauseCovariant", source)
}
