<?php

include 'config.php';

$dbconn = pg_connect("host=$db_hostname dbname=$db_database user=$db_username password=$db_password")
	or die('Could not connect: ' . pg_last_error());

// Check if the form has been submitted
if ($_SERVER['REQUEST_METHOD'] == 'POST') {

	// Get the form data
	$first_name = $_POST['first_name'];
	$last_name = $_POST['last_name'];
	$phone = $_POST['phone'];
	$email = $_POST['email'];
  
	// Insert the new record into the database
	$query = "INSERT INTO address_book (first_name, last_name, phone, email,) VALUES ('$first_name', '$last_name', '$phone', '$email')";
	$result = pg_query($dbconn, $query);
	if (!$result) {
	  die("Error: " . pg_last_error($dbconn));
	}
  }
  
  // Query the database for all records
  $query = "SELECT * FROM address_book ORDER BY last_name, first_name";
  $result = pg_query($dbconn, $query);
  if (!$result) {
	die("Error: " . pg_last_error($dbconn));
  }
  
  ?>
  
  <!-- HTML form for adding new records -->
  <form method="post" action="">
	<label for="first_name">First name:</label><br>
	<input type="text" id="first_name" name="first_name"><br>
	<label for="last_name">Last name:</label><br>
	<input type="text" id="last_name" name="last_name"><br>
	<label for="phone">Phone:</label><br>
	<input type="text" id="phone" name="phone"><br>
	<label for="email">Email:</label><br>
	<input type="text" id="email" name="email"><br>
	<input type="submit" value="Submit">
  </form>
  
  <!-- HTML table for displaying records -->
  <table>
	<tr>
	  <th>Name</th>
	  <th>Phone</th>
	  <th>Email</th>
	</tr>
	<?php while ($row = pg_fetch_array($result)) { ?>
	  <tr>
		<td><?php echo $row['first_name'] . ' ' . $row['last_name']; ?></td>
		<td><?php echo $row['phone']; ?></td>
		<td><?php echo $row['email']; ?></td>
	  </tr>
	<?php } ?>
  </table>
  
  <?php
  
  // Close the database connection
  pg_close($dbconn);
  
