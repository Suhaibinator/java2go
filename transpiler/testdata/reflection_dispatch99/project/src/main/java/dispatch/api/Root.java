package dispatch.api;
public interface Root { static String marker() { return "root-static"; } default String label() { return "root-default"; } }
