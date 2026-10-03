package dispatch.api;
public interface Root { static String marker() { return "root-static"; } default String extra() { return "root-extra"; } default String label() { return "root-default"; } }
