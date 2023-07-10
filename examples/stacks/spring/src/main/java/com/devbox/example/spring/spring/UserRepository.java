package com.devbox.example.spring.spring;

import org.springframework.data.repository.CrudRepository;
import com.devbox.example.spring.spring.User;

public interface UserRepository extends CrudRepository<User,
Integer>{
}
