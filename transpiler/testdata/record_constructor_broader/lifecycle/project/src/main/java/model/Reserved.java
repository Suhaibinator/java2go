package model;
public record Reserved(int type,int map,int func){public Reserved{type++;map+=2;func+=3;}}
