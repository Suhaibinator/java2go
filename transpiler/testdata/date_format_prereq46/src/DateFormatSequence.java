import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.text.ParseException;
import java.util.Date;
import java.util.Locale;
import java.util.TimeZone;

// Stateful repeated formatter operations matching the public API used by Gson.
public class DateFormatSequence {
    static void parseThenRestore(DateFormat format,String input){
        TimeZone saved=format.getTimeZone();
        try{
            Date parsed=format.parse(input);
            System.out.println("parsed="+input+":"+parsed.getTime()+":"+format.getTimeZone().getID());
        }catch(ParseException failure){System.out.println("parse.error="+failure.getErrorOffset()+":"+failure.getMessage());}
        finally{format.setTimeZone(saved);}
        System.out.println("restored="+format.getTimeZone().getID()+":"+format.format(new Date(0)));
    }
    public static void main(String[] args)throws Exception{
        Locale priorLocale=Locale.getDefault();TimeZone priorZone=TimeZone.getDefault();
        try{
            Locale.setDefault(Locale.US);TimeZone.setDefault(TimeZone.getTimeZone("UTC"));
            DateFormat captured=new SimpleDateFormat("yyyy-MM-dd HH:mm:ss z",Locale.US);
            DateFormat implicit=new SimpleDateFormat("MMM d, yyyy");
            TimeZone.setDefault(TimeZone.getTimeZone("GMT+05:30"));
            DateFormat later=new SimpleDateFormat("yyyy-MM-dd HH:mm:ss z",Locale.US);
            System.out.println("capture="+captured.format(new Date(0))+"|"+later.format(new Date(0)));
            System.out.println("pattern="+((SimpleDateFormat)captured).toPattern());
            parseThenRestore(captured,"1970-01-01 00:00:00 PST");
            parseThenRestore(captured,"1970-01-01 00:00:00 UTC trailing");
            parseThenRestore(captured,"not a date");
            parseThenRestore(implicit,"Feb 30, 2020");
            DateFormat clock=new SimpleDateFormat("hh:mm:ss a",Locale.US);clock.setTimeZone(TimeZone.getTimeZone("UTC"));
            parseThenRestore(clock,"12:00:00 AM");parseThenRestore(clock,"12:00:00 PM");
            for(int style:new int[]{DateFormat.SHORT,DateFormat.MEDIUM,DateFormat.LONG,DateFormat.FULL}){
                DateFormat explicit=DateFormat.getDateTimeInstance(style,style,Locale.US);explicit.setTimeZone(TimeZone.getTimeZone("UTC"));
                DateFormat defaults=DateFormat.getDateTimeInstance(style,style);defaults.setTimeZone(TimeZone.getTimeZone("UTC"));
                String text=explicit.format(new Date(1709251199000L));
                System.out.println("style="+style+":"+text+":"+defaults.format(new Date(1709251199000L))+":"+explicit.parse(text).getTime());
            }
            System.out.println("constant="+DateFormat.DEFAULT+":"+DateFormat.MEDIUM);
        }finally{Locale.setDefault(priorLocale);TimeZone.setDefault(priorZone);}
    }
}
