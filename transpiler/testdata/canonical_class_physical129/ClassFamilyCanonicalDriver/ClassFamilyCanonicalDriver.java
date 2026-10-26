package classproof;
public class ClassFamilyCanonicalDriver {
 @SuppressWarnings({"rawtypes","unchecked"}) public static void main(String[] args)throws Exception {
  CanonicalClassStorageOracle.main(args);
  Class<CanonicalClassStorageOracle.Child>[] array=(Class<CanonicalClassStorageOracle.Child>[])new Class<?>[]{CanonicalClassStorageOracle.Child.class};
  CanonicalClassStorageOracle.Slot<CanonicalClassStorageOracle.Child> slot=new CanonicalClassStorageOracle.Slot<>(CanonicalClassStorageOracle.Base.class,CanonicalClassStorageOracle.Child.class,array);
  CanonicalClassStorageOracle.Slot<?> erased=slot;CanonicalClassStorageOracle.Slot raw=slot;
  System.out.println("family-alias="+(erased==slot)+":"+(raw==slot)+":"+(raw.entries()==array));
  raw.put(String.class);raw.exact=Integer.class;raw.entries[0]=String.class;
  System.out.println("family-mutation="+((Object)slot.get()==String.class)+":"+((Object)slot.exact()==Integer.class)+":"+((Object)slot.entries()[0]==String.class));
  System.out.println("family-nominal="+(slot.entries().getClass().getComponentType()==Class.class)+":"+(slot.get().getClass()==Class.class));
 }
}
